package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ama31337/near-go-warchest/common"
	cmd "github.com/ama31337/near-go-warchest/helpers"
	"github.com/ama31337/near-go-warchest/rpc"
	prom "github.com/ama31337/near-go-warchest/services/prometheus"
)

type Runner struct {
	poolId, defaultDelegatorId                         string
	delegatorIds                                       []string
	expectedStake                                      int
	rpcSuccess, rpcFailed                              int
	delegatorStakedBalance, delegatorUnStakedBalance   map[string]int
	currentSeatPrice, nextSeatPrice, expectedSeatPrice int
}

func NewRunner(poolId string, delegatorIds []string) *Runner {
	var defaultDelegatorId string
	delegatorStakedBalance := make(map[string]int)
	delegatorUnStakedBalance := make(map[string]int)
	for _, delegatorId := range delegatorIds {
		delegatorStakedBalance[delegatorId] = 0
		delegatorUnStakedBalance[delegatorId] = 0
		defaultDelegatorId = delegatorId
	}
	return &Runner{
		poolId:                   poolId,
		delegatorIds:             delegatorIds,
		defaultDelegatorId:       defaultDelegatorId,
		delegatorStakedBalance:   delegatorStakedBalance,
		delegatorUnStakedBalance: delegatorUnStakedBalance,
	}
}

func (r *Runner) Run(ctx context.Context, resCh chan *rpc.SubscrResult, m *prom.PromMetrics, sem common.Sem) {

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	var notInProposals bool
	var epochStartHeight int64
	var leftBlocksPrev, estimatedBlocksCountPerReq int // per 90 sec
	for {
		select {
		case res := <-resCh:
			sem.Acquare()
			if res.Err != nil {
				r.rpcFailed++
				log.Println("Failed to connect to RPC")
				if r.rpcSuccess > 0 {
					log.Println("Using cache...")
					res.LatestBlockHeight = res.LatestBlockHeight + int64(estimatedBlocksCountPerReq)
					// Estimated new epoch
					if res.LatestBlockHeight >= res.EpochStartHeight+int64(res.EpochLength) {
						res.EpochStartHeight += int64(res.EpochLength)
					}
				} else {
					sem.Release()
					continue
				}
			}
			r.rpcSuccess++
			if epochStartHeight == 0 {
				epochStartHeight = res.EpochStartHeight
				leftBlocksPrev = int(res.EpochStartHeight) - int(res.LatestBlockHeight) + res.EpochLength
			}
			leftBlocks := int(res.EpochStartHeight) - int(res.LatestBlockHeight) + res.EpochLength
			estimatedBlocksCountPerReq = leftBlocksPrev - leftBlocks
			leftBlocksPrev = leftBlocks
			log.Printf("LatestBlockHeight: %d\n", res.LatestBlockHeight)
			log.Printf("EpochStartHeight: %d\n", res.EpochStartHeight)
			log.Printf("Left Blocks: %d\n", leftBlocks)

			r.expectedStake = getExpectedStake(ctx, r.poolId)
			if r.expectedStake != 0 {
				log.Printf("Expected stake: %d\n", r.expectedStake)
				notInProposals = false
				m.ExpectedStakeGauge.Set(float64(r.expectedStake))
			} else {
				log.Printf("You are not in proposals\n")
				notInProposals = true
			}
			log.Printf("Current stake: %d\n", res.CurrentStake)
			log.Printf("Next stake: %d\n", res.NextStake)

			// multiple delegator accounts
			var totalDelegatorsStakedBalance, totalDelegatorsUnStakedBalance int
			for _, delegatorId := range r.delegatorIds {
				dsb, err := getDelegatorStakedBalance(ctx, r.poolId, delegatorId)
				if err == nil {
					r.delegatorStakedBalance[delegatorId] = dsb
					totalDelegatorsStakedBalance += dsb
				}
				log.Printf("%s staked balance: %d\n", delegatorId, dsb)

				dusb, err := getDelegatorUnStakedBalance(ctx, r.poolId, delegatorId)
				if err == nil {
					r.delegatorUnStakedBalance[delegatorId] = dusb
					totalDelegatorsUnStakedBalance += dusb
				}
				log.Printf("%s unstaked balance: %d\n", delegatorId, dusb)
			}
			m.DStakedBalanceGauge.Set(float64(totalDelegatorsStakedBalance))
			m.DUnStakedBalanceGauge.Set(float64(totalDelegatorsUnStakedBalance))

			m.LeftBlocksGauge.Set(float64(leftBlocks))
			m.StakeAmountGauge.Set(float64(res.CurrentStake))
			m.RestakeGauge.Set(0)
			m.PingGauge.Set(0)

			if epochStartHeight != res.EpochStartHeight {
				// New epoch
				// If the new epoch then ping
				log.Println("Starting ping...")
				command := fmt.Sprintf(pingCmd, r.poolId, r.defaultDelegatorId)
				_, err := cmd.Run(ctx, command)
				if err != nil {
					m.PingGauge.Set(0)
				} else {
					log.Printf("Success: %s\n", command)
					epochStartHeight = res.EpochStartHeight
					if res.CurrentStake == 0 {
						m.PingGauge.Set(float64(100000))
					} else {
						m.PingGauge.Set(float64(res.CurrentStake))
					}
				}
			}
			if !r.fetchPrices(ctx, m.NextSeatPriceGauge, m.ExpectedSeatPriceGauge) {
				sem.Release()
				continue
			}

			if notInProposals || res.KickedOut {
				sem.Release()
				continue
			}

			// Seats calculation
			seats := float64(r.expectedStake) / float64(r.expectedSeatPrice)
			log.Printf("Expected seats: %f", seats)

			if leftBlocks > 1000 {
				log.Printf("Too early tp stake/unstake, left blocks = %d", leftBlocks)
			}

			if seats > 1.001 && leftBlocks < 1000 {
				log.Printf("You retain %f seats\n", seats)
				tokensAmountMap := getTokensAmountToRestake("unstake", r.delegatorStakedBalance, r.expectedStake, r.expectedSeatPrice)
				if len(tokensAmountMap) == 0 {
					log.Printf("You don't have enough staked balance\n")
					sem.Release()
					continue
				}
				// Run near unstake
				restake(ctx, r.poolId, "unstake", tokensAmountMap, m.RestakeGauge, m.StakeAmountGauge)
			} else if seats < 1.0 && leftBlocks < 1000 {
				log.Printf("You don't have enough stake to get one seat: %f\n", seats)
				tokensAmountMap := getTokensAmountToRestake("stake", r.delegatorUnStakedBalance, r.expectedStake, r.expectedSeatPrice)
				// Run near stake
				restake(ctx, r.poolId, "stake", tokensAmountMap, m.RestakeGauge, m.StakeAmountGauge)
			} else if seats >= 1.0 && seats < 1.001 {
				log.Println("I'm okay with a stake")
			}
			sem.Release()
		case <-ctx.Done():
			return
		case <-sigc:
			log.Println("System kill")
			os.Exit(0)
		}
	}
}

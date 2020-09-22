package rpc

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ama31337/near-go-warchest/common"
	nearapi "github.com/ama31337/near-go-warchest/rpc/client"
	prom "github.com/ama31337/near-go-warchest/services/prometheus"
)

var (
	repeatTime = os.Getenv("REPEAT_TIME")
)

type SubscrResult struct {
	LatestBlockHeight int64
	EpochStartHeight  int64
	EpochLength       int
	CurrentStake      int
	NextStake         int
	KickedOut         bool
	Err               error
}

type Monitor struct {
	client *nearapi.Client
	result *SubscrResult
	poolId string
}

func NewMonitor(client *nearapi.Client, poolId string) *Monitor {
	return &Monitor{
		client: client,
		poolId: poolId,
	}
}

func (m *Monitor) Run(ctx context.Context, result chan *SubscrResult, sem common.Sem, metrics *prom.PromMetrics) {
	t := common.GetIntFromString(repeatTime)
	ticker := time.NewTicker(time.Duration(t) * time.Second)
	log.Printf("Subscribed for updates every %s seconds\n", repeatTime)
	for {
		select {
		case <-ticker.C:
			sem.Acquare()

			log.Println("Starting watch rpc")

			sr, err := m.client.Get("status", nil)
			if err != nil {
				log.Println(err)
				sem.Release()
				m.result.Err = err
				result <- m.result
				continue
			}

			var epochLength int
			switch sr.Status.ChainId {
			case "betanet":
				epochLength = 10000
			case "testnet":
				epochLength = 43200
			case "mainnet":
				epochLength = 43200
			case "guildnet":
				epochLength = 43200
			}

			blockHeight := sr.Status.SyncInfo.LatestBlockHeight

			vr, err := m.client.Get("validators", []uint64{blockHeight})
			if err != nil {
				log.Println(err)
				sem.Release()
				m.result.Err = err
				result <- m.result
				continue
			}

			metrics.ThresholdGauge.Set(0)
			var currentStake int

			for _, v := range vr.Validators.CurrentValidators {

				if v.AccountId == m.poolId {

					pb := float64(v.NumProducedBlocks)
					eb := float64(v.NumExpectedBlocks)
					threshold := (pb / eb) * 100
					if threshold > 90.0 {
						log.Printf("Kicked out threshold: %f\n", threshold)
					}
					metrics.ThresholdGauge.Set(threshold)

					currentStake = common.GetStakeFromString(v.Stake)
				}
			}

			var nextStake int
			for _, v := range vr.Validators.NextValidators {
				if v.AccountId == m.poolId {
					nextStake = common.GetStakeFromString(v.Stake)
				}
			}

			kickedOut := false
			for _, v := range vr.Validators.PrevEpochKickOut {
				if v.AccountId == m.poolId {
					kickedOut = true
					log.Printf("Was kicked out :(\n")
				}
			}

			m.result = &SubscrResult{
				EpochStartHeight:  vr.Validators.EpochStartHeight,
				LatestBlockHeight: int64(blockHeight),
				EpochLength:       epochLength,
				CurrentStake:      currentStake,
				NextStake:         nextStake,
				KickedOut:         kickedOut,
				Err:               nil,
			}

			sem.Release()
			result <- m.result

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

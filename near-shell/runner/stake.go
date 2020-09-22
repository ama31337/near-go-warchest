package runner

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ama31337/near-go-warchest/common"
	cmd "github.com/ama31337/near-go-warchest/helpers"
	"github.com/prometheus/client_golang/prometheus"
)

func restake(ctx context.Context, poolId, method string, tokensAmountMap map[string]int, restakeGauge, stakeAmountGauge prometheus.Gauge) bool {
	if len(tokensAmountMap) == 0 {
		return false
	}
	for delegatorId, delegatorBalance := range tokensAmountMap {
		tokensAmountStr := common.GetStringFromStake(delegatorBalance)
		stakeAmountGauge.Set(float64(delegatorBalance))

		log.Printf("%s: Starting %s %d NEAR\n", delegatorId, method, delegatorBalance)
		err := runStake(ctx, poolId, method, tokensAmountStr, delegatorId)
		if err != nil {
			return false
		}
		log.Printf("%s: Success %sd %d NEAR\n", delegatorId, method, delegatorBalance)
		restakeGauge.Set(float64(delegatorBalance))
	}

	return true
}

func runStake(ctx context.Context, poolId, method, amount, delegatorId string) error {
	_, err := cmd.Run(ctx, fmt.Sprintf(stakeCmd, poolId, method, amount, delegatorId))
	log.Printf("Stake | unstake %s \n", fmt.Sprintf(stakeCmd, poolId, method, amount, delegatorId))
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func getExpectedStake(ctx context.Context, accountId string) int {
	currentProp, err := cmd.Run(ctx, fmt.Sprintf(proposalsCmd, accountId))
	if err != nil {
		log.Printf("Failed to run proposalsCmd, %s",fmt.Sprintf(proposalsCmd, accountId) )
		return 0
	}
	if currentProp != "" {
		sa := strings.Split(currentProp, "|")
		if len(sa) >= 4 {
			s := sa[3]
			if len(strings.Fields(s)) > 1 {
				return common.GetIntFromString(strings.Fields(s)[2])
			} else {
				return common.GetIntFromString(strings.Fields(s)[0])
			}
		}
	}
	return 0
}

type delegator struct {
	delegatorBalance int
	delegatorId      string
}

type entries []delegator

func (s entries) Len() int           { return len(s) }
func (s entries) Less(i, j int) bool { return s[i].delegatorBalance < s[j].delegatorBalance }
func (s entries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func getTokensAmountToRestake(method string, delegatorBalances map[string]int, expectedStake, expectedSeatPrice int) map[string]int {
	var delegatorBalancesSorted entries
	for k, v := range delegatorBalances {
		delegatorBalancesSorted = append(delegatorBalancesSorted, delegator{delegatorBalance: v, delegatorId: k})
	}

	sort.Sort(sort.Reverse(delegatorBalancesSorted))

	tokensAmountMap := make(map[string]int)
	var balances []int
	for _, v := range delegatorBalancesSorted {
		var tokensAmount int
		// Stake
		if method == "stake" {
			tokensAmount = expectedSeatPrice - expectedStake + 100
			var sumOfStake int
			if len(balances) > 0 {
				for _, v := range balances {
					sumOfStake += v
				}
				sumOfStake += v.delegatorBalance
				if sumOfStake > tokensAmount {
					overage := sumOfStake - tokensAmount
					if v.delegatorBalance-overage == 0 {
						return tokensAmountMap
					}
					tokensAmountMap[v.delegatorId] = v.delegatorBalance - overage
					return tokensAmountMap
				}
			}

			if tokensAmount > v.delegatorBalance {
				log.Printf("%s not enough balance to stake %d NEAR\n", v.delegatorId, tokensAmount)
				tokensAmountMap[v.delegatorId] = v.delegatorBalance
				balances = append(balances, v.delegatorBalance)
				continue
			}
			tokensAmountMap[v.delegatorId] = tokensAmount
			return tokensAmountMap
		} else {
			// Unstake
			offset := 100
			for tokensAmount < v.delegatorBalance-offset && expectedStake-tokensAmount > expectedSeatPrice+offset {
				tokensAmount += offset
			}
			if tokensAmount == 0 {
				break
			}
			tokensAmountMap[v.delegatorId] = tokensAmount
			expectedStake -= tokensAmount
		}
	}
	return tokensAmountMap
}

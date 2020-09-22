package runner

import (
	"context"
	"fmt"
	// "log"
	"github.com/ama31337/near-go-warchest/common"
	cmd "github.com/ama31337/near-go-warchest/helpers"
)

func getDelegatorStakedBalance(ctx context.Context, poolId, delegatorId string) (int, error) {
	r, err := cmd.Run(ctx, fmt.Sprintf(getStakedBalanceCmd, poolId, delegatorId))
		// log.Printf("getStakedBalanceCmd, %s",fmt.Sprintf(getStakedBalanceCmd, poolId, delegatorId) )

	if err != nil {
		return 0, err
	}
	return common.GetStakeFromNearView(r), nil
}

func getDelegatorUnStakedBalance(ctx context.Context, poolId, delegatorId string) (int, error) {
	r, err := cmd.Run(ctx, fmt.Sprintf(getUnStakedBalanceCmd, poolId, delegatorId))
		// log.Printf("getUnStakedBalanceCmd, %s",fmt.Sprintf(getUnStakedBalanceCmd, poolId, delegatorId) )

	if err != nil {
		return 0, err
	}
	return common.GetStakeFromNearView(r), nil
}

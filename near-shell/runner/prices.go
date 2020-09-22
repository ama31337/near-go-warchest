package runner

import (
	"context"
	"log"

	"github.com/ama31337/near-go-warchest/common"
	cmd "github.com/ama31337/near-go-warchest/helpers"
	"github.com/prometheus/client_golang/prometheus"
)

func (r *Runner) fetchPrices(ctx context.Context, nextSeatPriceGauge, expectedSeatPriceGauge prometheus.Gauge) bool {
	if r.currentSeatPrice == 0 {
		// Current seat price
		csp, err := getSeatPrice(ctx, currentSeatPriceCmd)
		if err != nil {
			log.Println("Failed to get currentSeatPrice")
			if r.currentSeatPrice == 0 {
				return false
			}
		} else {
			r.currentSeatPrice = csp
		}
		log.Printf("Current seat price %d\n", r.currentSeatPrice)
	}
	// Next seat price
	nsp, err := getSeatPrice(ctx, nextSeatPriceCmd)
	if err != nil {
		log.Println("Failed to get nextSeatPrice")
		if r.nextSeatPrice == 0 {
			return false
		}
	} else {
		r.nextSeatPrice = nsp
	}
	log.Printf("Next seat price %d\n", r.nextSeatPrice)
	nextSeatPriceGauge.Set(float64(r.nextSeatPrice))

	// Expected seat price
	esp, err := getSeatPrice(ctx, proposalsSeatPriceCmd)
	if err != nil {
		log.Println("Failed to get expectedSeatPrice")
		if r.expectedSeatPrice == 0 {
			return false
		}
	} else {
		r.expectedSeatPrice = esp
	}
	log.Printf("Expected seat price %d\n", r.expectedSeatPrice)
	expectedSeatPriceGauge.Set(float64(r.expectedSeatPrice))
	return true
}

func getSeatPrice(ctx context.Context, command string) (int, error) {
	r, err := cmd.Run(ctx, command)
	if err != nil {
		log.Printf("Failed to run %s", command)
		return 0, err
	}
	return common.GetIntFromString(r), nil
}

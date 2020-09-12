# Near Go Warchest Bot.

This tool/service dynamically maintaining no more than one seat and export metrics for [monitoring](https://prometheus.io). It uses [JSON-RPC](https://docs.near.org/docs/interaction/rpc) and [Near Shell](https://github.com/near/near-shell/) command line interface.

## Features

- Dynamically maintaining one seat 
- Supporting multi delegate accounts
- ping() with a new epoch
- Prometheus metrics
- Docker

## Install Docker
```
sudo apt-get update
sudo apt install docker.io
```

## Usage

### Docker

You can deploy using the [masknetgoal634/near-go-warchest](https://hub.docker.com/r/masknetgoal634/near-go-warchest) Docker image.

    git clone https://github.com/masknetgoal634/near-go-warchest

    cd near-go-warchest

Make sure you have a keys of your delegator account at `$HOME/.near-credential`.

```
sudo docker run -dti \
     --restart always \
     --volume $HOME/.near-credentials:/root/.near-credentials \
     --name go-warchest \
     --network=host \
     --env-file env.list \
     -p 9444:9444 \ 
     masknetgoal634/near-go-warchest:latest /dist/go-warchest -accountId <YOUR_POOL_ID> -delegatorId <YOUR_DELEGATOR1_ID> -delegatorId <YOUR_DELEGATOR2_ID>
```

To watch the logs: 

    sudo docker logs go-warchest -f

### Without Docker

Intall and/or uddate Go. You need to have 1.13 at least
https://medium.com/@khongwooilee/how-to-update-the-go-version-6065f5c8c3ec


You have to install [Near Shell](https://github.com/near/near-shell/).

Make sure you have a keys of your delegator account at `$HOME/.near-credential`.

    git clone https://github.com/masknetgoal634/near-go-warchest

    cd near-go-warchest

    set -a
    source env.list
    set +a

    go build go-warchest.go

    ./go-warchest -accountId <YOUR_POOL_ID> -delegatorId <YOUR_DELEGATOR_ID>


By default the near-go-warchest metrics service serves on `:9444` at `/metrics`.

## Prometheus

Here is an example `prometheus.yml`.

```
  - job_name: go-warchest
    scrape_interval: 30s
    static_configs:
    - targets: ['127.0.0.1:9444']
```
## Grafana

You can add alerts for the metrics listed below.

![](https://raw.githubusercontent.com/masknetgoal634/near-go-warchest/master/img/dashboard.png)

## Exported Metrics

| Name | Description |
| ---- | ----------- |
| warchest_left_blocks | The number of blocks left in the current epoch |
| warchest_ping | The Near shell ping event |
| warchest_restake | The Near shell restake event |
| warchest_stake_amount | The amount of stake |
| warchest_next_seat_price | The next seat price |
| warchest_expected_seat_price | The expected seat price |
| warchest_expected_stake | The expected stake |
| warchest_threshold | The kickout threshold (%) |
| warchest_delegator_staked_balance | The delegator staked balance |
| warchest_delegator_unstaked_balance | The delegator unstaked balance |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

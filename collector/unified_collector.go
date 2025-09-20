package collector

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"zerog-exporter/config"
	"zerog-exporter/rpc"
	"zerog-exporter/util"
)

// Simple logger for the collector
type Logger struct{}

// convertFromBaseUnit converts from base unit (e.g., 36000000000) to display unit (e.g., 36)
func convertFromBaseUnit(baseAmount int64, decimals int) float64 {
	if decimals == 0 {
		return float64(baseAmount)
	}
	return float64(baseAmount) / math.Pow10(decimals)
}

// convertFromBaseUnitFloat converts from base unit using float64 to handle large numbers
func convertFromBaseUnitFloat(baseAmount float64, decimals int) float64 {
	if decimals == 0 {
		return baseAmount
	}
	return baseAmount / math.Pow10(decimals)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	// Simple logging - can be enhanced later
}

func (l *Logger) Error(msg string, args ...interface{}) {
	// Simple logging - can be enhanced later
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	// Simple logging - can be enhanced later
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	// Simple logging - can be enhanced later
}

// UnifiedCollector collects metrics from both Cosmos SDK and Ethereum
type UnifiedCollector struct {
	client              *rpc.Client
	cfg                 *config.Chain
	ethereumConfig      *config.Ethereum
	prometheusServer    string
	logger              *Logger
	blocksBehind        float64
	blockTimeCalculator *util.BlockTimeCalculator
	validatorStates     map[string]*validatorState

	// General Metrics
	cosmosBlockTime     *prometheus.Desc
	cosmosAvgBlockTime  *prometheus.Desc
	cosmosTimeSinceLastBlock *prometheus.Desc

	// Supply & Pool Metrics
	bondedTokens        *prometheus.Desc
	notBondedTokens     *prometheus.Desc
	communityPool       *prometheus.Desc
	supplyTotal         *prometheus.Desc
	inflation           *prometheus.Desc
	annualProvisions    *prometheus.Desc

	// Wallet Metrics
	walletBalance       *prometheus.Desc
	walletDelegations   *prometheus.Desc
	walletRewards       *prometheus.Desc
	walletUnbonding     *prometheus.Desc

	// Validator Metrics
	validatorTokens     *prometheus.Desc
	validatorCommissionRate *prometheus.Desc
	validatorCommission *prometheus.Desc
	validatorRewards    *prometheus.Desc
	validatorMissedBlocks *prometheus.Desc
	validatorRank       *prometheus.Desc
	validatorActive     *prometheus.Desc
	validatorStatus     *prometheus.Desc
	validatorJailedDesc *prometheus.Desc
	validatorDelegatorShares *prometheus.Desc

	// Validator Statistics
	validatorsTotal     *prometheus.Desc
	validatorsActive    *prometheus.Desc
	validatorsInactive  *prometheus.Desc
	validatorsBondedRatio *prometheus.Desc

	// Chain Parameters
	paramsSignedBlocksWindow *prometheus.Desc
	paramsMinSignedPerWindow *prometheus.Desc
	paramsDowntimeJailDuration *prometheus.Desc
	paramsSlashFractionDoubleSign *prometheus.Desc
	paramsSlashFractionDowntime *prometheus.Desc
	paramsMaxValidators *prometheus.Desc
	paramsBaseProposerReward *prometheus.Desc
	paramsBonusProposerReward *prometheus.Desc

	// Governance Metrics
	consensusProposalChain *prometheus.Desc
	consensusProposalReceiveCount *prometheus.Desc

	// Tenderduty Metrics
	tdUp                *prometheus.Desc
	tdNodeHeight        *prometheus.Desc
	tdBlocksBehind      *prometheus.Desc
	tdSignedBlocks     *prometheus.Desc
	tdMissedBlocks     *prometheus.Desc
	tdConsecutiveMissed *prometheus.Desc
	tdValidatorActive   *prometheus.Desc
	tdValidatorJailed   *prometheus.Desc
	tdTimeSinceLastBlock *prometheus.Desc

	// Ethereum Metrics
	ethBlockNumber      *prometheus.Desc
	ethValidatorBalance *prometheus.Desc
	ethStakingContract  *prometheus.Desc
	ethTotalValidators  *prometheus.Desc
	ethActiveValidators *prometheus.Desc
	ethStakingPool      *prometheus.Desc
	ethMaxValidators    *prometheus.Desc
	ethValidatorCount   *prometheus.Desc
}

type validatorState struct {
	consensusAddress string
	moniker          string
	status           int
	active           bool
	jailed           bool
}

// NewUnifiedCollector creates a new UnifiedCollector
func NewUnifiedCollector(client *rpc.Client, cfg *config.Chain, ethereumConfig *config.Ethereum, prometheusServer string) *UnifiedCollector {
	return &UnifiedCollector{
		client:              client,
		cfg:                 cfg,
		ethereumConfig:      ethereumConfig,
		prometheusServer:    prometheusServer,
		logger:              &Logger{},
		blockTimeCalculator: util.NewBlockTimeCalculator(100),
		validatorStates:     make(map[string]*validatorState),

		// General Metrics
		cosmosBlockTime: prometheus.NewDesc("cosmos_block_time", "Last block time", []string{"chain_id"}, nil),
		cosmosAvgBlockTime: prometheus.NewDesc("cosmos_avg_block_time", "Average block time", []string{"chain_id"}, nil),
		cosmosTimeSinceLastBlock: prometheus.NewDesc("cosmos_time_since_last_block", "Time since last block", []string{"chain_id"}, nil),

		// Supply & Pool Metrics
		bondedTokens: prometheus.NewDesc("cosmos_bonded_tokens", "Bonded tokens", []string{"chain_id", "denom"}, nil),
		notBondedTokens: prometheus.NewDesc("cosmos_not_bonded_tokens", "Not bonded tokens", []string{"chain_id", "denom"}, nil),
		communityPool: prometheus.NewDesc("cosmos_community_pool", "Community pool balance", []string{"chain_id", "denom"}, nil),
		supplyTotal: prometheus.NewDesc("cosmos_supply_total", "Total supply", []string{"chain_id", "denom"}, nil),
		inflation: prometheus.NewDesc("cosmos_inflation", "Inflation rate", []string{"chain_id"}, nil),
		annualProvisions: prometheus.NewDesc("cosmos_annual_provisions", "Annual provisions", []string{"chain_id", "denom"}, nil),

		// Wallet Metrics
		walletBalance: prometheus.NewDesc("cosmos_wallet_balance", "Wallet balance", []string{"chain_id", "address", "denom"}, nil),
		walletDelegations: prometheus.NewDesc("cosmos_wallet_delegations", "Wallet delegations", []string{"chain_id", "address", "denom"}, nil),
		walletRewards: prometheus.NewDesc("cosmos_wallet_rewards", "Wallet rewards", []string{"chain_id", "address", "denom"}, nil),
		walletUnbonding: prometheus.NewDesc("cosmos_wallet_unbonding", "Wallet unbonding", []string{"chain_id", "address", "denom"}, nil),

		// Validator Metrics
		validatorTokens: prometheus.NewDesc("cosmos_validator_tokens", "Validator tokens", []string{"chain_id", "address", "moniker", "denom"}, nil),
		validatorCommissionRate: prometheus.NewDesc("cosmos_validator_commission_rate", "Validator commission rate", []string{"chain_id", "address", "moniker"}, nil),
		validatorCommission: prometheus.NewDesc("cosmos_validator_commission", "Validator commission", []string{"chain_id", "address", "moniker", "denom"}, nil),
		validatorRewards: prometheus.NewDesc("cosmos_validator_rewards", "Validator rewards", []string{"chain_id", "address", "moniker", "denom"}, nil),
		validatorMissedBlocks: prometheus.NewDesc("cosmos_validator_missed_blocks", "Validator missed blocks", []string{"chain_id", "address", "moniker"}, nil),
		validatorRank: prometheus.NewDesc("cosmos_validators_rank", "Validator rank", []string{"chain_id", "address", "moniker"}, nil),
		validatorActive: prometheus.NewDesc("cosmos_validator_active", "Validator active status", []string{"chain_id", "address", "moniker"}, nil),
		validatorStatus: prometheus.NewDesc("cosmos_validator_status", "Validator status", []string{"chain_id", "address", "moniker"}, nil),
		validatorJailedDesc: prometheus.NewDesc("cosmos_validator_jailed_status", "Validator jailed status", []string{"chain_id", "address", "moniker"}, nil),
		validatorDelegatorShares: prometheus.NewDesc("cosmos_validators_delegator_shares", "Validator delegator shares", []string{"chain_id", "address", "moniker"}, nil),

		// Validator Statistics
		validatorsTotal: prometheus.NewDesc("cosmos_validators_total", "Total validators", []string{"chain_id"}, nil),
		validatorsActive: prometheus.NewDesc("cosmos_validators_active", "Active validators", []string{"chain_id"}, nil),
		validatorsInactive: prometheus.NewDesc("cosmos_validators_inactive", "Inactive validators", []string{"chain_id"}, nil),
		validatorsBondedRatio: prometheus.NewDesc("cosmos_validators_bonded_ratio", "Bonded ratio", []string{"chain_id"}, nil),

		// Chain Parameters
		paramsSignedBlocksWindow: prometheus.NewDesc("cosmos_params_signed_blocks_window", "Signed blocks window", []string{"chain_id"}, nil),
		paramsMinSignedPerWindow: prometheus.NewDesc("cosmos_params_min_signed_per_window", "Min signed per window", []string{"chain_id"}, nil),
		paramsDowntimeJailDuration: prometheus.NewDesc("cosmos_params_downtime_jail_duration", "Downtime jail duration", []string{"chain_id"}, nil),
		paramsSlashFractionDoubleSign: prometheus.NewDesc("cosmos_params_slash_fraction_double_sign", "Slash fraction double sign", []string{"chain_id"}, nil),
		paramsSlashFractionDowntime: prometheus.NewDesc("cosmos_params_slash_fraction_downtime", "Slash fraction downtime", []string{"chain_id"}, nil),
		paramsMaxValidators: prometheus.NewDesc("cosmos_params_max_validators", "Max validators", []string{"chain_id"}, nil),
		paramsBaseProposerReward: prometheus.NewDesc("cosmos_params_base_proposer_reward", "Base proposer reward", []string{"chain_id"}, nil),
		paramsBonusProposerReward: prometheus.NewDesc("cosmos_params_bonus_proposer_reward", "Bonus proposer reward", []string{"chain_id"}, nil),

		// Governance Metrics
		consensusProposalChain: prometheus.NewDesc("cometbft_consensus_proposal_chain", "Consensus proposal chain", []string{"chain_id"}, nil),
		consensusProposalReceiveCount: prometheus.NewDesc("cosmos_consensus_proposal_receive_count", "Consensus proposal receive count", []string{"chain_id", "status"}, nil),

		// Tenderduty Metrics
		tdUp: prometheus.NewDesc("cosmos_td_up", "Tenderduty status", []string{"chain_id"}, nil),
		tdNodeHeight: prometheus.NewDesc("cosmos_td_node_height", "Tenderduty node height", []string{"chain_id"}, nil),
		tdBlocksBehind: prometheus.NewDesc("cosmos_td_blocks_behind", "Tenderduty blocks behind", []string{"chain_id"}, nil),
		tdSignedBlocks: prometheus.NewDesc("cosmos_td_signed_blocks", "Tenderduty signed blocks", []string{"chain_id"}, nil),
		tdMissedBlocks: prometheus.NewDesc("cosmos_validators_missed_blocks", "Validators missed blocks", []string{"chain_id"}, nil),
		tdConsecutiveMissed: prometheus.NewDesc("cosmos_td_consecutive_missed", "Tenderduty consecutive missed", []string{"chain_id"}, nil),
		tdValidatorActive: prometheus.NewDesc("cosmos_td_validator_active", "Tenderduty validator active", []string{"chain_id"}, nil),
		tdValidatorJailed: prometheus.NewDesc("cosmos_td_validator_jailed", "Tenderduty validator jailed", []string{"chain_id"}, nil),
		tdTimeSinceLastBlock: prometheus.NewDesc("cosmos_td_time_since_last_block", "Tenderduty time since last block", []string{"chain_id"}, nil),

		// Ethereum Metrics
		ethBlockNumber: prometheus.NewDesc("eth_block_number", "Ethereum block number", []string{"chain_id"}, nil),
		ethValidatorBalance: prometheus.NewDesc("eth_validator_balance", "Validator balance on Ethereum", []string{"chain_id", "address", "moniker"}, nil),
		ethStakingContract: prometheus.NewDesc("eth_staking_contract", "Staking contract status", []string{"chain_id", "contract"}, nil),
		ethTotalValidators: prometheus.NewDesc("eth_total_validators", "Total validators on contract", []string{"chain_id"}, nil),
		ethActiveValidators: prometheus.NewDesc("eth_active_validators", "Active validators on contract", []string{"chain_id"}, nil),
		ethStakingPool: prometheus.NewDesc("eth_staking_pool", "Staking pool balance", []string{"chain_id"}, nil),
		ethMaxValidators: prometheus.NewDesc("eth_max_validators", "Maximum validators", []string{"chain_id"}, nil),
		ethValidatorCount: prometheus.NewDesc("eth_validator_count", "Validator count", []string{"chain_id"}, nil),
	}
}

// Describe implements prometheus.Collector
func (c *UnifiedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cosmosBlockTime
	ch <- c.cosmosAvgBlockTime
	ch <- c.cosmosTimeSinceLastBlock
	ch <- c.bondedTokens
	ch <- c.notBondedTokens
	ch <- c.communityPool
	ch <- c.supplyTotal
	ch <- c.inflation
	ch <- c.annualProvisions
	ch <- c.walletBalance
	ch <- c.walletDelegations
	ch <- c.walletRewards
	ch <- c.walletUnbonding
	ch <- c.validatorTokens
	ch <- c.validatorCommissionRate
	ch <- c.validatorCommission
	ch <- c.validatorRewards
	ch <- c.validatorMissedBlocks
	ch <- c.validatorRank
	ch <- c.validatorActive
	ch <- c.validatorStatus
	ch <- c.validatorJailedDesc
	ch <- c.validatorDelegatorShares
	ch <- c.validatorsTotal
	ch <- c.validatorsActive
	ch <- c.validatorsInactive
	ch <- c.validatorsBondedRatio
	ch <- c.paramsSignedBlocksWindow
	ch <- c.paramsMinSignedPerWindow
	ch <- c.paramsDowntimeJailDuration
	ch <- c.paramsSlashFractionDoubleSign
	ch <- c.paramsSlashFractionDowntime
	ch <- c.paramsMaxValidators
	ch <- c.paramsBaseProposerReward
	ch <- c.paramsBonusProposerReward
	ch <- c.consensusProposalChain
	ch <- c.consensusProposalReceiveCount
	ch <- c.tdSignedBlocks
	ch <- c.tdMissedBlocks
	ch <- c.tdConsecutiveMissed
	ch <- c.tdValidatorActive
	ch <- c.tdValidatorJailed
	ch <- c.tdTimeSinceLastBlock
	ch <- c.ethBlockNumber
	ch <- c.ethValidatorBalance
	ch <- c.ethStakingContract
	ch <- c.ethTotalValidators
	ch <- c.ethActiveValidators
	ch <- c.ethStakingPool
	ch <- c.ethMaxValidators
	ch <- c.ethValidatorCount
}

// Collect implements prometheus.Collector
func (c *UnifiedCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error { return c.collectCosmosMetrics(ctx, ch) })
	g.Go(func() error { return c.collectEthereumMetrics(ch) })

	if err := g.Wait(); err != nil {
		c.logger.Error("Error collecting metrics", "error", err)
	}
}

// collectCosmosMetrics collects metrics from Cosmos SDK
func (c *UnifiedCollector) collectCosmosMetrics(ctx context.Context, ch chan<- prometheus.Metric) error {
	// Get node status
	status, err := c.client.GetStatus()
	if err != nil {
		c.logger.Error("Failed to get node status", "error", err)
		return err
	}

	// Block time metrics (using current time since LatestBlockTime is not available)
	currentTime := time.Now()
	ch <- prometheus.MustNewConstMetric(c.cosmosBlockTime, prometheus.GaugeValue, float64(currentTime.Unix()), c.cfg.ChainID)
	
	// Update block time calculator
	if height, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
		c.blockTimeCalculator.UpdateBlockTime(height, currentTime)
	}
	
	// Average block time
	if avgBlockTime := c.blockTimeCalculator.GetAverageBlockTime(); avgBlockTime > 0 {
		ch <- prometheus.MustNewConstMetric(c.cosmosAvgBlockTime, prometheus.GaugeValue, avgBlockTime.Seconds(), c.cfg.ChainID)
	}
	
	// Time since last block
	if timeSinceLastBlock := c.blockTimeCalculator.GetLatestBlockTime(); timeSinceLastBlock > 0 {
		ch <- prometheus.MustNewConstMetric(c.cosmosTimeSinceLastBlock, prometheus.GaugeValue, timeSinceLastBlock.Seconds(), c.cfg.ChainID)
	}

	// Validator statistics from latest block signatures
	activeValidators := 0
	inactiveValidators := 0
	totalValidators := 0
	
	if currentHeight, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
		if block, err := c.client.GetBlock(int(currentHeight)); err == nil {
			// block_id_flag 분석
			// 1 = Precommit (이전 블록 서명)
			// 4 = Commit (현재 블록 서명) - Active
			// 5 = Absent (서명 안됨) - Inactive
			for _, sig := range block.Result.Block.LastCommit.Signatures {
				totalValidators++
				if sig.BlockIDFlag == 4 {
					activeValidators++
				} else if sig.BlockIDFlag == 5 {
					inactiveValidators++
				}
			}
		}
	}

	// Validator statistics
	ch <- prometheus.MustNewConstMetric(c.validatorsTotal, prometheus.GaugeValue, float64(totalValidators), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.validatorsActive, prometheus.GaugeValue, float64(activeValidators), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.validatorsInactive, prometheus.GaugeValue, float64(inactiveValidators), c.cfg.ChainID)
	
	// Bonded ratio 계산
	bondedRatio := 0.0
	if totalValidators > 0 {
		bondedRatio = float64(activeValidators) / float64(totalValidators)
	}
	ch <- prometheus.MustNewConstMetric(c.validatorsBondedRatio, prometheus.GaugeValue, bondedRatio, c.cfg.ChainID)
	

	


	// Supply & Pool metrics - 실제 API 호출로 데이터 수집
	if stakingPool, err := c.client.GetStakingPool(); err == nil {
		if bondedTokens, err := strconv.ParseInt(stakingPool.Pool.BondedTokens, 10, 64); err == nil {
			bondedTokensFloat := convertFromBaseUnit(bondedTokens, c.cfg.TokenDecimals)
			ch <- prometheus.MustNewConstMetric(c.bondedTokens, prometheus.GaugeValue, bondedTokensFloat, c.cfg.ChainID, "0G")
		}
		if notBondedTokens, err := strconv.ParseInt(stakingPool.Pool.NotBondedTokens, 10, 64); err == nil {
			notBondedTokensFloat := convertFromBaseUnit(notBondedTokens, c.cfg.TokenDecimals)
			ch <- prometheus.MustNewConstMetric(c.notBondedTokens, prometheus.GaugeValue, notBondedTokensFloat, c.cfg.ChainID, "0G")
		}
	}

	// Community Pool
	if communityPool, err := c.client.GetCommunityPool(); err == nil {
		for _, pool := range communityPool.Pool {
			if amount, err := strconv.ParseInt(pool.Amount, 10, 64); err == nil {
				amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
				ch <- prometheus.MustNewConstMetric(c.communityPool, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, pool.Denom)
			}
		}
	}

	// Bank Supply
	if bankSupply, err := c.client.GetBankSupply(); err == nil {
		for _, supply := range bankSupply.Supply {
			if amount, err := strconv.ParseInt(supply.Amount, 10, 64); err == nil {
				amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
				ch <- prometheus.MustNewConstMetric(c.supplyTotal, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, supply.Denom)
			}
		}
	}

	// Inflation
	if inflation, err := c.client.GetMintingInflation(); err == nil {
		if inflationRate, err := strconv.ParseFloat(inflation.Inflation, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.inflation, prometheus.GaugeValue, inflationRate, c.cfg.ChainID)
		}
	}

	// Annual Provisions
	if annualProvisions, err := c.client.GetMintingAnnualProvisions(); err == nil {
		if provisions, err := strconv.ParseInt(annualProvisions.AnnualProvisions, 10, 64); err == nil {
			provisionsFloat := convertFromBaseUnit(provisions, c.cfg.TokenDecimals)
			ch <- prometheus.MustNewConstMetric(c.annualProvisions, prometheus.GaugeValue, provisionsFloat, c.cfg.ChainID, "0G")
		}
	}

	// Wallet metrics - 실제 API 호출로 데이터 수집
	for _, wallet := range c.cfg.Wallets {
		// Wallet Balance
		if balance, err := c.client.GetWalletBalance(wallet.Address); err == nil {
			for _, bal := range balance.Balances {
				if amount, err := strconv.ParseInt(bal.Amount, 10, 64); err == nil {
					amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
					ch <- prometheus.MustNewConstMetric(c.walletBalance, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, wallet.Address, bal.Denom)
				}
			}
		}

		// Wallet Delegations
		if delegations, err := c.client.GetWalletDelegations(wallet.Address); err == nil {
			for _, del := range delegations.DelegationResponses {
				if amount, err := strconv.ParseInt(del.Balance.Amount, 10, 64); err == nil {
					amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
					ch <- prometheus.MustNewConstMetric(c.walletDelegations, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, wallet.Address, del.Balance.Denom)
				}
			}
		}

		// Wallet Rewards
		if rewards, err := c.client.GetWalletRewards(wallet.Address); err == nil {
			for _, reward := range rewards.Rewards {
				for _, r := range reward.Reward {
					if amount, err := strconv.ParseInt(r.Amount, 10, 64); err == nil {
						amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
						ch <- prometheus.MustNewConstMetric(c.walletRewards, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, wallet.Address, r.Denom)
					}
				}
			}
		}

		// Wallet Unbonding
		if unbonding, err := c.client.GetWalletUnbonding(wallet.Address); err == nil {
			for _, ub := range unbonding.UnbondingResponses {
				for _, entry := range ub.Entries {
					if amount, err := strconv.ParseInt(entry.Balance, 10, 64); err == nil {
						amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
						ch <- prometheus.MustNewConstMetric(c.walletUnbonding, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, wallet.Address, "0G")
					}
				}
			}
		}
	}

	// Chain parameters - 실제 API 호출로 데이터 수집
	// Slashing Parameters
	if slashingParams, err := c.client.GetSlashingParams(); err == nil {
		if signedBlocksWindow, err := strconv.ParseInt(slashingParams.Params.SignedBlocksWindow, 10, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsSignedBlocksWindow, prometheus.GaugeValue, float64(signedBlocksWindow), c.cfg.ChainID)
		}
		if minSignedPerWindow, err := strconv.ParseFloat(slashingParams.Params.MinSignedPerWindow, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsMinSignedPerWindow, prometheus.GaugeValue, minSignedPerWindow, c.cfg.ChainID)
		}
		if downtimeJailDuration, err := strconv.ParseFloat(slashingParams.Params.DowntimeJailDuration, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsDowntimeJailDuration, prometheus.GaugeValue, downtimeJailDuration, c.cfg.ChainID)
		}
		if slashFractionDoubleSign, err := strconv.ParseFloat(slashingParams.Params.SlashFractionDoubleSign, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsSlashFractionDoubleSign, prometheus.GaugeValue, slashFractionDoubleSign, c.cfg.ChainID)
		}
		if slashFractionDowntime, err := strconv.ParseFloat(slashingParams.Params.SlashFractionDowntime, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsSlashFractionDowntime, prometheus.GaugeValue, slashFractionDowntime, c.cfg.ChainID)
		}
	}

	// Staking Parameters
	if stakingParams, err := c.client.GetStakingParams(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.paramsMaxValidators, prometheus.GaugeValue, float64(stakingParams.Params.MaxValidators), c.cfg.ChainID)
	}

	// Distribution Parameters
	if distributionParams, err := c.client.GetDistributionParams(); err == nil {
		if baseProposerReward, err := strconv.ParseFloat(distributionParams.Params.BaseProposerReward, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsBaseProposerReward, prometheus.GaugeValue, baseProposerReward, c.cfg.ChainID)
		}
		if bonusProposerReward, err := strconv.ParseFloat(distributionParams.Params.BonusProposerReward, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.paramsBonusProposerReward, prometheus.GaugeValue, bonusProposerReward, c.cfg.ChainID)
		}
	}

	// Governance metrics - 실제 API 호출로 데이터 수집
	if proposals, err := c.client.GetGovernanceProposals(); err == nil {
		proposalCounts := make(map[string]int)
		for _, proposal := range proposals.Proposals {
			proposalCounts[proposal.Status]++
		}
		
		for status, count := range proposalCounts {
			ch <- prometheus.MustNewConstMetric(c.consensusProposalReceiveCount, prometheus.GaugeValue, float64(count), c.cfg.ChainID, status)
		}
	}

	// Tenderduty metrics - 실제 블록 분석 기반
	// 최근 100개 블록에서 signing 정보 분석
	signedBlocks := 0
	missedBlocks := 0
	maxConsecutiveMissed := 0
	
	// Config의 모든 validator 주소들에 대해 분석
	validatorStats := make(map[string]struct {
		signedBlocks     int
		missedBlocks     int
		consecutiveMissed int
		maxConsecutiveMissed int
		proposals        int
	})
	
	// 초기화
	for _, validatorAddr := range c.cfg.Validators {
		validatorStats[validatorAddr] = struct {
			signedBlocks     int
			missedBlocks     int
			consecutiveMissed int
			maxConsecutiveMissed int
			proposals        int
		}{}
	}
	
	if currentHeight, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
		for i := int64(0); i < 100 && currentHeight-i > 0; i++ {
			if block, err := c.client.GetBlock(int(currentHeight - i)); err == nil {
				// Proposal 확인
				proposerAddr := block.Result.Block.Header.ProposerAddress
				if stats, exists := validatorStats[proposerAddr]; exists {
					stats.proposals++
					validatorStats[proposerAddr] = stats
				}
				
				// 각 validator의 서명 확인
				for validatorAddr := range validatorStats {
					hasSigned := false
					for _, sig := range block.Result.Block.LastCommit.Signatures {
						if sig.ValidatorAddress == validatorAddr {
							// block_id_flag: 4 = Commit (서명됨), 5 = Absent (서명 안됨)
							if sig.BlockIDFlag == 4 {
								hasSigned = true
								break
							}
						}
					}
					
					stats := validatorStats[validatorAddr]
					if hasSigned {
						stats.signedBlocks++
						stats.consecutiveMissed = 0
					} else {
						stats.missedBlocks++
						stats.consecutiveMissed++
						if stats.consecutiveMissed > stats.maxConsecutiveMissed {
							stats.maxConsecutiveMissed = stats.consecutiveMissed
						}
					}
					validatorStats[validatorAddr] = stats
				}
			}
		}
	}
	
	// 전체 통계 계산 (첫 번째 validator 기준, 비활성 시 0으로 설정)
	if len(c.cfg.Validators) > 0 {
		firstValidator := c.cfg.Validators[0]
		if stats, exists := validatorStats[firstValidator]; exists {
			signedBlocks = stats.signedBlocks
			missedBlocks = stats.missedBlocks
			maxConsecutiveMissed = stats.maxConsecutiveMissed
			
			// 첫 번째 validator의 active status 확인
			validatorActive := 0.0
			if currentHeight, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
				if block, err := c.client.GetBlock(int(currentHeight)); err == nil {
					for _, sig := range block.Result.Block.LastCommit.Signatures {
						if sig.ValidatorAddress == firstValidator {
							if sig.BlockIDFlag == 4 {
								validatorActive = 1.0
							} else if sig.BlockIDFlag == 5 {
								validatorActive = 0.0
							}
							break
						}
					}
				}
			}
			
			// 비활성 validator의 경우 missed blocks를 0으로 설정
			if validatorActive == 0.0 {
				missedBlocks = 0
				maxConsecutiveMissed = 0
			}
		}
	}
	
	// Tenderduty metrics
	height := int64(0)
	if h, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
		height = h
	}
	ch <- prometheus.MustNewConstMetric(c.tdUp, prometheus.GaugeValue, 1, c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdNodeHeight, prometheus.GaugeValue, float64(height), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdBlocksBehind, prometheus.GaugeValue, 0, c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdSignedBlocks, prometheus.GaugeValue, float64(signedBlocks), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdMissedBlocks, prometheus.GaugeValue, float64(missedBlocks), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdConsecutiveMissed, prometheus.GaugeValue, float64(maxConsecutiveMissed), c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdValidatorActive, prometheus.GaugeValue, 1, c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdValidatorJailed, prometheus.GaugeValue, 0, c.cfg.ChainID)
	ch <- prometheus.MustNewConstMetric(c.tdTimeSinceLastBlock, prometheus.GaugeValue, 0, c.cfg.ChainID)
	
	// 각 validator별 개별 메트릭 생성 - 실제 API 호출로 데이터 수집
	// 먼저 모든 밸리데이터 정보를 가져옴
	validators, err := c.client.GetValidators()
	if err != nil {
		c.logger.Error("Failed to get validators", "error", err)
		return err
	}

	// 밸리데이터 정보를 맵으로 저장
	validatorInfoMap := make(map[string]struct {
		Moniker          string
		Tokens           string
		DelegatorShares  string
		CommissionRate   string
		Status           string
		Jailed           bool
		ConsensusAddress string
	})

	for _, validator := range validators.Validators {
		validatorInfoMap[validator.ConsensusAddress] = struct {
			Moniker          string
			Tokens           string
			DelegatorShares  string
			CommissionRate   string
			Status           string
			Jailed           bool
			ConsensusAddress string
		}{
			Moniker:          validator.Description.Moniker,
			Tokens:           validator.Tokens,
			DelegatorShares:  validator.DelegatorShares,
			CommissionRate:   validator.Commission.CommissionRates.Rate,
			Status:           validator.Status,
			Jailed:           validator.Jailed,
			ConsensusAddress: validator.ConsensusAddress,
		}
	}

	for validatorAddr, stats := range validatorStats {
		// Validator active status (block_id_flag 기반)
		validatorActive := 0.0
		if currentHeight, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
			if block, err := c.client.GetBlock(int(currentHeight)); err == nil {
				for _, sig := range block.Result.Block.LastCommit.Signatures {
					if sig.ValidatorAddress == validatorAddr {
						if sig.BlockIDFlag == 4 {
							validatorActive = 1.0
						} else if sig.BlockIDFlag == 5 {
							validatorActive = 0.0
						}
						break
					}
				}
			}
		}
		
		// 비활성 validator의 경우 서명/미스블럭을 0으로 설정
		missedBlocks := stats.missedBlocks
		if validatorActive == 0.0 {
			missedBlocks = 0
		}
		
		// 밸리데이터 정보 가져오기
		var moniker string = "Unknown"
		var tokens string = "0"
		var delegatorShares string = "0"
		var commissionRate string = "0"
		var validatorStatus string = "UNBONDED"
		var jailed bool = false
		
		if info, exists := validatorInfoMap[validatorAddr]; exists {
			moniker = info.Moniker
			tokens = info.Tokens
			delegatorShares = info.DelegatorShares
			commissionRate = info.CommissionRate
			validatorStatus = info.Status
			jailed = info.Jailed
		}
		
		// Missed blocks 메트릭
		ch <- prometheus.MustNewConstMetric(c.validatorMissedBlocks, prometheus.GaugeValue, float64(missedBlocks), c.cfg.ChainID, validatorAddr, moniker)
		ch <- prometheus.MustNewConstMetric(c.validatorActive, prometheus.GaugeValue, validatorActive, c.cfg.ChainID, validatorAddr, moniker)
		
		// Validator 토큰 및 위임량
		if tokensInt, err := strconv.ParseInt(tokens, 10, 64); err == nil {
			tokensFloat := convertFromBaseUnit(tokensInt, c.cfg.TokenDecimals)
			ch <- prometheus.MustNewConstMetric(c.validatorTokens, prometheus.GaugeValue, tokensFloat, c.cfg.ChainID, validatorAddr, moniker, "0G")
		}
		
		if delegatorSharesFloat, err := strconv.ParseFloat(delegatorShares, 64); err == nil {
			delegatorSharesConverted := convertFromBaseUnitFloat(delegatorSharesFloat, c.cfg.TokenDecimals)
			ch <- prometheus.MustNewConstMetric(c.validatorDelegatorShares, prometheus.GaugeValue, delegatorSharesConverted, c.cfg.ChainID, validatorAddr, moniker)
		}
		
		// Commission Rate
		if commissionRateFloat, err := strconv.ParseFloat(commissionRate, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.validatorCommissionRate, prometheus.GaugeValue, commissionRateFloat, c.cfg.ChainID, validatorAddr, moniker)
		}
		
		// Commission 및 Rewards (실제 API 호출)
		if commission, err := c.client.GetValidatorCommission(validatorAddr); err == nil {
			for _, comm := range commission.Commission.Commission {
				if amount, err := strconv.ParseInt(comm.Amount, 10, 64); err == nil {
					amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
					ch <- prometheus.MustNewConstMetric(c.validatorCommission, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, validatorAddr, moniker, comm.Denom)
				}
			}
		}
		
		if rewards, err := c.client.GetValidatorRewards(validatorAddr); err == nil {
			for _, reward := range rewards.Rewards.Rewards {
				if amount, err := strconv.ParseInt(reward.Amount, 10, 64); err == nil {
					amountFloat := convertFromBaseUnit(amount, c.cfg.TokenDecimals)
					ch <- prometheus.MustNewConstMetric(c.validatorRewards, prometheus.GaugeValue, amountFloat, c.cfg.ChainID, validatorAddr, moniker, reward.Denom)
				}
			}
		}
		
		// Status 및 Jailed
		var statusValue float64
		switch validatorStatus {
		case "BOND_STATUS_BONDED":
			statusValue = 3
		case "BOND_STATUS_UNBONDING":
			statusValue = 2
		case "BOND_STATUS_UNBONDED":
			statusValue = 1
		default:
			statusValue = 0
		}
		
		var jailedValue float64 = 0
		if jailed {
			jailedValue = 1
		}
		
		ch <- prometheus.MustNewConstMetric(c.validatorRank, prometheus.GaugeValue, 0, c.cfg.ChainID, validatorAddr, moniker)
		ch <- prometheus.MustNewConstMetric(c.validatorStatus, prometheus.GaugeValue, statusValue, c.cfg.ChainID, validatorAddr, moniker)
		ch <- prometheus.MustNewConstMetric(c.validatorJailedDesc, prometheus.GaugeValue, jailedValue, c.cfg.ChainID, validatorAddr, moniker)
	}
	
	// 전체 proposal 수 계산 (첫 번째 validator 기준)
	if len(c.cfg.Validators) > 0 {
		firstValidator := c.cfg.Validators[0]
		if stats, exists := validatorStats[firstValidator]; exists {
			ch <- prometheus.MustNewConstMetric(c.consensusProposalChain, prometheus.GaugeValue, float64(stats.proposals), c.cfg.ChainID)
		}
	}

	return nil
}

// collectEthereumMetrics collects metrics from Ethereum JSON-RPC
func (c *UnifiedCollector) collectEthereumMetrics(ch chan<- prometheus.Metric) error {
	// Only collect Ethereum metrics for 0G Galileo Testnet
	if c.cfg.ChainID != "0g-galileo-testnet" {
		return nil
	}

	// Create Ethereum client
	var ethClient *util.EthereumClient
	if c.ethereumConfig != nil && c.ethereumConfig.JWTSecret != "" {
		ethClient = util.NewEthereumClientWithJWT(c.ethereumConfig.RPCURL, c.ethereumConfig.JWTSecret)
		c.logger.Info("Using Ethereum RPC with JWT authentication")
			} else {
		ethClient = util.NewEthereumClient(c.ethereumConfig.RPCURL)
		c.logger.Warn("Using Ethereum RPC without JWT authentication")
	}

	// Ethereum block number
	if blockNumber, err := ethClient.GetBlockNumber(); err == nil {
		if blockNum, err := strconv.ParseInt(blockNumber[2:], 16, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.ethBlockNumber, prometheus.GaugeValue, float64(blockNum), c.cfg.ChainID)
		}
	} else {
		c.logger.Error("Failed to get Ethereum block number", "error", err)
	}

	// Staking contract status
	stakingContract := c.ethereumConfig.StakingContract
	if _, err := ethClient.GetBalance(stakingContract); err == nil {
		ch <- prometheus.MustNewConstMetric(c.ethStakingContract, prometheus.GaugeValue, 1, c.cfg.ChainID, stakingContract)
	} else {
		ch <- prometheus.MustNewConstMetric(c.ethStakingContract, prometheus.GaugeValue, 0, c.cfg.ChainID, stakingContract)
		c.logger.Error("Failed to get staking contract status", "error", err)
	}

	// Ethereum addresses balance
	for _, ethAddr := range c.ethereumConfig.EthereumAddresses {
		if balance, err := ethClient.GetBalance(ethAddr.Address); err == nil {
			if bal, err := strconv.ParseInt(balance[2:], 16, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(c.ethValidatorBalance, prometheus.GaugeValue, float64(bal), c.cfg.ChainID, ethAddr.Address, "unknown")
			}
					} else {
			c.logger.Error("Failed to get Ethereum address balance", "address", ethAddr.Address, "error", err)
		}
	}

	// Contract-based metrics (these may fail due to incorrect function selectors)
	if totalValidators, err := ethClient.GetTotalValidators(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.ethTotalValidators, prometheus.GaugeValue, float64(totalValidators), c.cfg.ChainID)
		c.logger.Info("Retrieved total validators", "count", totalValidators)
				} else {
		c.logger.Error("Failed to get total validators", "error", err)
	}

	if activeValidators, err := ethClient.GetActiveValidators(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.ethActiveValidators, prometheus.GaugeValue, float64(activeValidators), c.cfg.ChainID)
		c.logger.Info("Retrieved active validators", "count", activeValidators)
			} else {
		c.logger.Error("Failed to get active validators", "error", err)
	}

	if stakingPool, err := ethClient.GetStakingPool(); err == nil {
		if poolBalance, err := strconv.ParseInt(stakingPool[2:], 16, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(c.ethStakingPool, prometheus.GaugeValue, float64(poolBalance), c.cfg.ChainID)
			c.logger.Info("Retrieved staking pool", "balance", poolBalance)
		}
					} else {
		c.logger.Error("Failed to get staking pool", "error", err)
	}

	if validatorCount, err := ethClient.GetValidatorCount(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.ethValidatorCount, prometheus.GaugeValue, float64(validatorCount), c.cfg.ChainID)
		c.logger.Info("Retrieved validator count", "count", validatorCount)
	} else {
		c.logger.Error("Failed to get validator count", "error", err)
	}

	if maxValidators, err := ethClient.GetMaxValidatorCount(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.ethMaxValidators, prometheus.GaugeValue, float64(maxValidators), c.cfg.ChainID)
		c.logger.Info("Retrieved max validators", "max", maxValidators)
	} else {
		c.logger.Error("Failed to get max validators", "error", err)
	}

	return nil
}

// updateValidatorMonikers updates validator moniker information
func (c *UnifiedCollector) updateValidatorMonikers(monikers map[string]string) {
	for addr, moniker := range monikers {
		if _, exists := c.validatorStates[addr]; !exists {
			c.validatorStates[addr] = &validatorState{
				consensusAddress: addr,
				moniker:          moniker,
				status:           3, // BOND_STATUS_BONDED
				active:           true,
				jailed:           false,
			}
		} else {
			c.validatorStates[addr].moniker = moniker
		}
	}
}

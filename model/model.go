package model

import randao "findora/randao/contract"

type Config struct {
	Http_listen string `json:"http_listen"`
	// CampaginIdsPath string `json:"campagin_ids_path"`
	Chain struct {
		Name        string `json:"name"`
		ChainId     string `json:"chain_id"`
		Endpoint    string `json:"endpoint"`
		EvEndpoint  string `json:"event_endpoint"`
		Participant string `json:"participant"`
		Campaigner  string `json:"campaigner"`
		Randao      string `json:"randao"`
		Opts        struct {
			GasLimit        string `json:"gas_limit"`
			MaxGasPrice     string `json:"max_gas_price"`
			MinGasReserve   string `json:"min_gas_reserve"`
			MaxDeposit      uint64 `json:"max_deposit"`
			MinRateOfReturn uint64 `json:"min_rate_of_return"`
			MinRevealWindow uint64 `json:"min_reveal_window"`
			MaxRevealDelay  uint64 `json:"max_reveal_delay"`
			MaxCampaigns    uint64 `json:"max_campaigns"`
			StartBlock      uint64 `json:"start_block"`
		} `json:"opts"`
	} `json:"chain"`
}

type CampaignInfo struct {
	Bnum           string `json:"bnum"`
	Deposit        string `json:"deposit"`
	CommitBalkline uint16 `json:"commitBalkline"`
	CommitDeadline uint16 `json:"commitDeadline"`
	Random         string `json:"random"`
	Settled        bool   `json:"settled"`
	Bountypot      string `json:"bountypot"`
	CommitNum      uint32 `json:"commitNum"`
	RevealsNum     uint32 `json:"revealsNum"`
}

func CampaignInfoConvert(campaignInfo *randao.IRandaoCampaignInfo) (dest *CampaignInfo) {
	dest = &CampaignInfo{
		Bnum:           campaignInfo.Bnum.String(),
		Deposit:        campaignInfo.Deposit.String(),
		CommitBalkline: campaignInfo.CommitBalkline,
		CommitDeadline: campaignInfo.CommitDeadline,
		Random:         campaignInfo.Random.String(),
		Settled:        campaignInfo.Settled,
		Bountypot:      campaignInfo.Bountypot.String(),
		CommitNum:      campaignInfo.CommitNum,
		RevealsNum:     campaignInfo.RevealsNum,
	}

	return
}

func CampaignInfoDeConvert(src *CampaignInfo) (dest *randao.IRandaoCampaignInfo) {
	dest = &randao.IRandaoCampaignInfo{}

	return
}

var Conf Config

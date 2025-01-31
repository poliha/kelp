package plugins

import (
	"fmt"
	"strings"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"github.com/stellar/kelp/support/utils"
)

// sdexFeed represents a pricefeed from the SDEX
type sdexFeed struct {
	sdex       *SDEX
	assetBase  *hProtocol.Asset
	assetQuote *hProtocol.Asset
}

// ensure that it implements PriceFeed
var _ api.PriceFeed = &sdexFeed{}

// makeSDEXFeed creates a price feed from buysell's url fields
func makeSDEXFeed(url string) (*sdexFeed, error) {
	urlParts := strings.Split(url, "/")

	baseAsset, e := parseHorizonAsset(urlParts[0])
	if e != nil {
		return nil, fmt.Errorf("unable to convert base asset url to sdex asset: %s", e)
	}
	quoteAsset, e := parseHorizonAsset(urlParts[1])
	if e != nil {
		return nil, fmt.Errorf("unable to convert quote asset url to sdex asset: %s", e)
	}

	tradingPair := &model.TradingPair{
		Base:  model.Asset(utils.Asset2CodeString(*baseAsset)),
		Quote: model.Asset(utils.Asset2CodeString(*quoteAsset)),
	}
	sdexAssetMap := map[model.Asset]hProtocol.Asset{
		tradingPair.Base:  *baseAsset,
		tradingPair.Quote: *quoteAsset,
	}

	var api *horizonclient.Client
	var ieif *IEIF
	var network build.Network
	if privateSdexHackVar != nil {
		api = privateSdexHackVar.API
		ieif = privateSdexHackVar.Ieif
		network = privateSdexHackVar.Network
	} else {
		// use production network by default
		api = horizonclient.DefaultPublicNetClient
		ieif = MakeIEIF(true)
		network = build.PublicNetwork
	}

	sdex := MakeSDEX(
		api,
		ieif,
		nil,
		"",
		"",
		"",
		"",
		network,
		nil,
		0,
		0,
		true,
		tradingPair,
		sdexAssetMap,
		SdexFixedFeeFn(0),
	)

	return &sdexFeed{
		sdex:       sdex,
		assetBase:  baseAsset,
		assetQuote: quoteAsset,
	}, nil
}

func parseHorizonAsset(assetString string) (*hProtocol.Asset, error) {
	parts := strings.Split(assetString, ":")
	code := parts[0]
	issuer := parts[1]

	asset, e := utils.ParseAsset(code, issuer)
	if e != nil {
		return nil, fmt.Errorf("could not read horizon asset from string (%s): %s", assetString, e)
	}

	return asset, e
}

// GetPrice returns the SDEX mid price for the trading pair
func (s *sdexFeed) GetPrice() (float64, error) {
	orderBook, e := s.sdex.GetOrderBook(s.sdex.pair, 1)
	if e != nil {
		return 0, fmt.Errorf("unable to get sdex price: %s", e)
	}

	topBidPrice := orderBook.Bids()[0].Price
	topAskPrice := orderBook.Asks()[0].Price

	centerPrice := topBidPrice.Add(*topAskPrice).Scale(0.5).AsFloat()
	return centerPrice, nil
}

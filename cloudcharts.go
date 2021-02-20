package cloudcharts

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

/*Quotes ...
 */
type Quotes struct {
	DefaultChartInterval string          `json:"_default_chart_interval,omitempty"`
	RefPrice             float64         `json:"_ref_price,omitempty"`
	Bars                 [][]interface{} `json:"_d"`
}

/*Candle ...
 */
type Candle struct {
	Time string
	// BOSSA API is OHLC go-echarts OCLH
	OHLC [4]float64
}

var (
	apiURL      = os.Getenv("API_URL")
	asset       = os.Getenv("ASSET")
	city        = os.Getenv("CITY")
	location, _ = time.LoadLocation(city)
	// http.Clients should be reused instead of created as needed.
	client = &http.Client{
		Timeout: 5 * time.Second,
	}
	userAgent = randUserAgent()
	kd        [100]Candle
)

func init() {
}

func main() {
}

/*CloudCharts ...
 */
func CloudCharts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	timeframe := ""
	if a := query.Get("a"); a != "" {
		asset = a
	}
	if t := query.Get("t"); t != "" {
		timeframe = t
	}
	if quotes := getQuotes(asset, timeframe); quotes != nil {
		for i, bar := range quotes.Bars {
			var tmp Candle
			tm := int64(bar[0].(float64))
			time := time.Unix(0, tm*int64(time.Millisecond))
			tmp.Time = time.In(location).Format("Jan _2 15:04")
			o, _ := bar[1].(float64)
			h, _ := bar[2].(float64)
			l, _ := bar[3].(float64)
			c, _ := bar[4].(float64)
			tmp.OHLC = [4]float64{o, c, l, h} // OHLC to OCLH
			kd[i] = tmp
		}
		kline := ohlcChart()
		kline.Render(w)
	} else {
		http.Error(w, "Something went wrong, can't render chart", http.StatusInternalServerError)
	}
}

func getQuotes(asset string, timeframe string) *Quotes {
	var quotes Quotes
	apiURL := fmt.Sprintf("%s%s.", apiURL, asset)
	if timeframe != "" {
		apiURL += "/" + timeframe
	}
	request, _ := http.NewRequest("GET", apiURL, nil)
	request.Header.Set("User-Agent", userAgent)
	if response, err := client.Do(request); err == nil {
		if err := json.NewDecoder(response.Body).Decode(&quotes); err != nil {
			log.Fatal(err)
		}
	}
	return &quotes
}

func ohlcChart() *charts.Kline {
	// create a new chart instance
	kline := charts.NewKLine()
	x := make([]string, 100)
	y := make([]opts.KlineData, 100)
	for i := 0; i < len(kd); i++ {
		x[i] = kd[i].Time
		y[i] = opts.KlineData{Value: kd[i].OHLC}
	}
	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    asset,
			Subtitle: fmt.Sprintf("%.5g", kd[99].OHLC[3]),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 20,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Scale: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "slider",
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show:  true,
			Right: "20%",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  true,
					Type:  "png",
					Title: "save as image",
				},
			}},
		),
	)
	kline.SetXAxis(x).AddSeries(asset, y).
		SetSeriesOptions(
			charts.WithMarkPointNameTypeItemOpts(opts.MarkPointNameTypeItem{
				Name:     "Maximum",
				Type:     "max",
				ValueDim: "highest",
			}),
			charts.WithMarkPointNameTypeItemOpts(opts.MarkPointNameTypeItem{
				Name:     "Minimum",
				Type:     "min",
				ValueDim: "lowest",
			}),
			charts.WithMarkPointStyleOpts(opts.MarkPointStyle{
				Symbol: []string{"pin"},
				Label: &opts.Label{
					Show: true,
				},
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color:        "#00da3c",
				Color0:       "#ec0000",
				BorderColor:  "#008F28",
				BorderColor0: "#8A0000",
			}),
		)
	return kline
}

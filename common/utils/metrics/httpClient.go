package metrics

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bentoml/yatai/common/utils/stats"
)

const SOLARCITY_ROUTE = "solarcity"
const SOLEOCUSTOMER_DESIGN_ROUTE = "HourlySystemProduction"
const POWERGATE_ASSET_PROXY_SITES_ROUTE = "powergate-asset-proxy/sites/"

type snapshot struct {
	state map[string]interface{}
	mu    *sync.Mutex
}

type HTTPClient struct {
	MetricsIdentifier string
	*http.Client
}

var Snapshot snapshot

func init() {
	Snapshot = snapshot{mu: new(sync.Mutex), state: make(map[string]interface{})}
	var getSetTime = 100 * time.Millisecond
	var ticker = time.NewTicker(getSetTime)
	var stateChannel = stats.StatsCollector.PollState(getSetTime)

	go func() {
		for range ticker.C {
			stats.StatsCollector.SetValueAtFieldPath(stats.StatsCollector.GetRuntimeStats(), "Runtime")
		}
	}()

	go func() {
		for state := range stateChannel {
			Snapshot.Set(state)
		}
	}()
}

func (snapshot *snapshot) Get() map[string]interface{} {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()

	return snapshot.state
}

func (snapshot *snapshot) Set(state map[string]interface{}) {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()

	snapshot.state = state
}

// Analytics collected for Grafana
func (mc *HTTPClient) Do(request *http.Request) (response *http.Response, err error) {
	// Start Wait timer
	var start = time.Now()

	// Perform Request
	response, err = mc.Client.Do(request)

	// Construct metric meta data
	var URLSlice []string
	var reportURL string
	var host string
	var metricsPath string

	// MetricsIdentifier used to more easily identify the client, as opposed to using the host
	// @todo PWG-1828 - update all clients to specify a MetricsIdentifier
	if mc.MetricsIdentifier != "" {
		host = mc.MetricsIdentifier
	} else {
		// Unique URL for querying metrics
		// Note: We want this to ideally be the same between our common data sources so we can view in one dashboard
		host = request.URL.Host

		// (QUICK HACK) TO ALIGN METRICS ON GRAFANA DASHBOARD
		// Grafana currently only queries PROD endpoints on its panels
		// NOT AN ACCURATE REPRESENTATION OF UNIQUE ROUTE URLs in Grafana
		// if strings.Contains(host, "api.eng.sn.xxx.services") {
		// 	host = strings.Replace(host, "eng.", "", 1)
		// } else if strings.Contains(host, "eng.sn.xxx.services") {
		// 	host = strings.Replace(host, "eng.", "prd.", 1)
		// }
	}

	// clients can also add a path to context that will be used to report metrics
	// @todo PWG-1828 - update all clients to specify a metrics-path
	if metricsPath = getMetricsPathFromContext(request.Context()); metricsPath != "" {
		reportURL = host + metricsPath
	} else {
		var URL = host + request.URL.Path

		switch URLSliceLen := len(strings.Split(URL, "/")); URLSliceLen >= 2 {
		case strings.Contains(URL, SOLARCITY_ROUTE) && !strings.Contains(URL, SOLEOCUSTOMER_DESIGN_ROUTE):
			fallthrough
		// Legacy solar-city config admin resource route
		case strings.Contains(URL, "solar_configs"):
			fallthrough
		// xxx PV Inverter & Wall Connector config admin resource route
		case strings.Contains(URL, "xxx_site_configs"):
			fallthrough
		case strings.Contains(URL, "dc_configs"):
			fallthrough
		// Config admin update DIN resource URL
		case strings.Contains(URL, "update_config_setting"):
			URLSlice = strings.Split(URL, "/")
			URLSlice = URLSlice[:URLSliceLen-1]
			reportURL = strings.Join(URLSlice, "/")
		// Asset API Too sites resource URL
		case (strings.Contains(URL, "sites/") || strings.Contains(URL, "hierarchies/")) && !strings.Contains(URL, POWERGATE_ASSET_PROXY_SITES_ROUTE):
			URLSlice = strings.Split(URL, "/")
			URLSlice = URLSlice[:URLSliceLen-1]
			// create unique reportURL to identify difference between /sites?din= and /sites/{uuid}
			reportURL = strings.Join(URLSlice, "/") + "/"
		case strings.Contains(URL, "configs") && !strings.Contains(URL, "powergate_info"):
			URLSlice = strings.Split(URL, "/")
			var popIndex int
			// when getting mangled din lists we need remove all elements after configs
			for i, v := range URLSlice {
				if v == "configs" {
					popIndex = i + 1
				}
			}
			if popIndex > 0 && popIndex < URLSliceLen {
				URLSlice = URLSlice[:popIndex]
			}
			reportURL = strings.Join(URLSlice, "/")
		case strings.Contains(URL, SOLEOCUSTOMER_DESIGN_ROUTE):
			// i.e. remove 0052bf56062247f7be4ca1c209707934 from
			// https://soleocustomerapi.xxx.com/api/v1/Design/0052bf56062247f7be4ca1c209707934/HourlySystemProduction
			// handled in case below
			fallthrough
		case strings.Contains(URL, POWERGATE_ASSET_PROXY_SITES_ROUTE):
			// i.e. remove 9dbe9a04-92b3-4576-a177-a03fa5f94d7d from
			// https://api.eng.sn.xxx.services/powergate-asset-proxy/sites/9dbe9a04-92b3-4576-a177-a03fa5f94d7d/programs
			// handled in case below
			fallthrough
		case strings.Contains(URL, "powergate_info"):
			URLSlice = strings.Split(URL, "/")
			URLSlice = append(URLSlice[:URLSliceLen-2], URLSlice[URLSliceLen-1])
			reportURL = strings.Join(URLSlice, "/")
		default:
			// we can still constantly grow if unique identifiers are in path
			reportURL = URL
		}
	}

	// Report stats to Influx
	stats.StatsCollector.AddUInt64AtFieldPath(uint64(time.Since(start)), "Service", "Outbound", request.Method+" "+reportURL, "Wait")
	stats.StatsCollector.IncrementUInt64AtFieldPath("Service", "Outbound", request.Method+" "+reportURL, "Count")
	if response != nil {
		stats.StatsCollector.IncrementUInt64AtFieldPath("Service", "Outbound", request.Method+" "+reportURL, "ResponseCodes", strconv.Itoa(response.StatusCode))
		// log success/failure based on status code
		if response.StatusCode > 299 {
			stats.StatsCollector.IncrementUInt64AtFieldPath("Service", "Outbound", request.Method+" "+reportURL, "ResponseCodes", "Fail")
		} else {
			stats.StatsCollector.IncrementUInt64AtFieldPath("Service", "Outbound", request.Method+" "+reportURL, "ResponseCodes", "Success")
		}
	} else {
		stats.StatsCollector.IncrementUInt64AtFieldPath("Service", "Outbound", request.Method+" "+reportURL, "Errors")
	}

	return
}

type metricsCtxKey string

const MetricsPathCtxID metricsCtxKey = "metrics-path"

func getMetricsPathFromContext(ctx context.Context) string {
	val, ok := ctx.Value(MetricsPathCtxID).(string)
	if !ok {
		return ""
	}
	return val
}

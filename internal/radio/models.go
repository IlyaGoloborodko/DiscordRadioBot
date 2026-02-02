package radio

type Station struct {
	Name      string `json:"name"`
	StreamURL string `json:"url_resolved"`
	Homepage  string `json:"homepage,omitempty"`
	Country   string `json:"country,omitempty"`
	Tags      string `json:"tags,omitempty"`
	Bitrate   int    `json:"bitrate,omitempty"`
}

var RecentSearch = make(map[string][]Station)

var AllStations []Station

package innertube

// Song represents a simplified song item for frontend consumption
type Song struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	Album        string `json:"album,omitempty"`
	Duration     int    `json:"duration"` // seconds
	DurationText string `json:"durationText,omitempty"`
	Thumbnail    string `json:"thumbnail"`
	IsTopic      bool   `json:"isTopic,omitempty"`
}

// StreamInfo represents the resolved audio stream information
type StreamInfo struct {
	VideoID   string `json:"videoId"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	StreamURL string `json:"streamUrl"`
	ProxyURL  string `json:"proxyUrl"`
	MimeType  string `json:"mimeType"`
	Bitrate   int    `json:"bitrate"`
	ExpiresIn int    `json:"expiresIn"`
}

// Internal Innertube request / response structures

type ClientContext struct {
	Client struct {
		ClientName    string `json:"clientName"`
		ClientVersion string `json:"clientVersion"`
		DeviceModel   string `json:"deviceModel,omitempty"`
		AndroidSDKVer int    `json:"androidSdkVersion,omitempty"`
		Hl            string `json:"hl"`
		Gl            string `json:"gl"`
	} `json:"client"`
}

type PlayerRequest struct {
	Context ClientContext `json:"context"`
	VideoID string        `json:"videoId"`
}

type SearchRequest struct {
	Context ClientContext `json:"context"`
	Query   string        `json:"query"`
	Params  string        `json:"params,omitempty"`
}

type PlayerResponse struct {
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	} `json:"playabilityStatus"`
	VideoDetails struct {
		VideoID       string `json:"videoId"`
		Title         string `json:"title"`
		LengthSeconds string `json:"lengthSeconds"`
		ChannelID     string `json:"channelId"`
		Author        string `json:"author"`
		Thumbnail     struct {
			Thumbnails []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
	} `json:"videoDetails"`
	StreamingData struct {
		ExpiresInSeconds string   `json:"expiresInSeconds"`
		Formats          []Format `json:"formats"`
		AdaptiveFormats  []Format `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

type Format struct {
	Itag             int    `json:"itag"`
	URL              string `json:"url"`
	MimeType         string `json:"mimeType"`
	Bitrate          int    `json:"bitrate"`
	AverageBitrate   int    `json:"averageBitrate,omitempty"`
	ContentLength    string `json:"contentLength,omitempty"`
	AudioSampleRate  string `json:"audioSampleRate,omitempty"`
	AudioQuality     string `json:"audioQuality,omitempty"`
	ApproxDurationMs string `json:"approxDurationMs,omitempty"`
}

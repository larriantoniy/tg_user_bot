package domain

// DefaultReaderBody add info https://ocr.space/OCRAPI
type DefaultReaderBody struct {
	Base64Image string `json:"base64Image"`
	Language    string `json:"language"`
}

type ReaderResponse struct {
	ParsedResults                []ParsedResult `json:"ParsedResults"`
	OCRExitCode                  string         `json:"OCRExitCode"`
	IsErroredOnProcessing        bool           `json:"IsErroredOnProcessing"`
	ErrorMessage                 *string        `json:"ErrorMessage"`
	ErrorDetails                 *string        `json:"ErrorDetails"`
	SearchablePDFURL             *string        `json:"SearchablePDFURL"`
	ProcessingTimeInMilliseconds string         `json:"ProcessingTimeInMilliseconds"`
}

type ParsedResult struct {
	TextOverlay       *TextOverlay `json:"TextOverlay"`
	FileParseExitCode int          `json:"FileParseExitCode"`
	ParsedText        *string      `json:"ParsedText"`
	ErrorMessage      *string      `json:"ErrorMessage"`
	ErrorDetails      *string      `json:"ErrorDetails"`
}

type TextOverlay struct {
	Lines      []Line  `json:"Lines"`
	HasOverlay bool    `json:"HasOverlay"`
	Message    *string `json:"Message"`
}

type Line struct {
	Words     []Word `json:"Words"`
	MaxHeight int    `json:"MaxHeight"`
	MinTop    int    `json:"MinTop"`
}

type Word struct {
	WordText string `json:"WordText"`
	Left     int    `json:"Left"`
	Top      int    `json:"Top"`
	Height   int    `json:"Height"`
	Width    int    `json:"Width"`
}

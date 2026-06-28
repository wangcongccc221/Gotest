package tcp

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

const (
	webSocketTopicIpmImage        = "ipmimage"
	cTCPSpliceImageInfoWireSize   = 52
	cTCPSimpleImageInfoWireSize   = 24
	cTCPMaxIpmImagePreviewBytes   = 64
	cTCPMaxIpmImageTailBytes      = 32
	cTCPSpliceImagePixelBitGray   = 1
	cTCPSpliceImagePixelBitYUV422 = 2
	cTCPSimpleImageFormatMono     = 1
	cTCPSimpleImageFormatYUV422   = 2
)

type cTCPIpmImageFrame struct {
	Timestamp       int64   `json:"timestamp"`
	PortName        string  `json:"portName"`
	CmdID           int32   `json:"cmdId"`
	CmdName         string  `json:"cmdName"`
	SrcID           int32   `json:"srcId"`
	DstID           int32   `json:"dstId"`
	PayloadLength   int     `json:"payloadLength"`
	ImageDataLength int     `json:"imageDataLength"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	RouteID         int     `json:"routeId"`
	FlawArea        uint32  `json:"flawArea,omitempty"`
	FlawNum         uint32  `json:"flawNum,omitempty"`
	PixelBit        uint8   `json:"pixelBit,omitempty"`
	ImageFormat     uint8   `json:"imageFormat,omitempty"`
	ColorCutY       float32 `json:"colorCutY,omitempty"`
	PreviewHex      string  `json:"previewHex,omitempty"`
	TailHex         string  `json:"tailHex,omitempty"`
	ImageDataBase64 string  `json:"imageDataBase64,omitempty"`
	BinDataBase64   string  `json:"binDataBase64,omitempty"`
	PngBase64       string  `json:"pngBase64,omitempty"`
	PngError        string  `json:"pngError,omitempty"`
}

type cTCPSpliceImageInfoWire struct {
	ImageDataLength int32
	Width           int32
	Height          int32
	RouteID         int32
	ChannelH        [cTCPServerMaxCameraDirection]int32
	ValidCupNum     int32
	ColorMidLen     int32
	FlawArea        uint32
	FlawNum         uint32
	ColorCutY       float32
	PixelBit        uint8
}

type cTCPSimpleImageInfoWire struct {
	ImageDataLength int32
	Width           int32
	Height          int32
	RouteID         int32
	Verify          uint32
	ImageFormat     uint8
}

const cTCPServerMaxCameraDirection = 3

func publishCTCPIpmImageFrame(serverName string, head cTCPServerCommandHead, payload []byte) error {
	frame, err := buildCTCPIpmImageFrame(serverName, head, payload)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal ipm image frame: %w", err)
	}
	return PublishWebSocketJSON(webSocketTopicIpmImage, string(raw))
}

func buildCTCPIpmImageFrame(serverName string, head cTCPServerCommandHead, payload []byte) (cTCPIpmImageFrame, error) {
	frame := cTCPIpmImageFrame{
		Timestamp:     cTCPNow().UnixMilli(),
		PortName:      serverName,
		CmdID:         head.NCmdId,
		CmdName:       cTCPCommandName(head.NCmdId),
		SrcID:         head.NSrcId,
		DstID:         head.NDstId,
		PayloadLength: len(payload),
	}

	switch head.NCmdId {
	case cmdIPMImageSplice, cmdIPMImageSpot:
		info, imageData, binData, err := parseCTCPSpliceImagePayload(payload)
		if err != nil {
			return frame, err
		}
		frame.ImageDataLength = int(info.ImageDataLength)
		frame.Width = int(info.Width)
		frame.Height = int(info.Height)
		frame.RouteID = int(info.RouteID)
		frame.FlawArea = info.FlawArea
		frame.FlawNum = info.FlawNum
		frame.PixelBit = info.PixelBit
		frame.ColorCutY = info.ColorCutY
		frame.PreviewHex = bytesToHexPreview(imageData, cTCPMaxIpmImagePreviewBytes)
		frame.TailHex = bytesToHexTail(imageData, cTCPMaxIpmImageTailBytes)
		frame.ImageDataBase64 = base64.StdEncoding.EncodeToString(imageData)
		if len(binData) > 0 {
			frame.BinDataBase64 = base64.StdEncoding.EncodeToString(binData)
		}
		pngBytes, err := encodeCTCPSpliceImagePNG(info, imageData)
		if err != nil {
			frame.PngError = err.Error()
		} else {
			frame.PngBase64 = base64.StdEncoding.EncodeToString(pngBytes)
		}
	case cmdIPMImage:
		info, imageData, err := parseCTCPSimpleImagePayload(payload)
		if err != nil {
			return frame, err
		}
		frame.ImageDataLength = int(info.ImageDataLength)
		frame.Width = int(info.Width)
		frame.Height = int(info.Height)
		frame.RouteID = int(info.RouteID)
		frame.ImageFormat = info.ImageFormat
		frame.PreviewHex = bytesToHexPreview(imageData, cTCPMaxIpmImagePreviewBytes)
		frame.TailHex = bytesToHexTail(imageData, cTCPMaxIpmImageTailBytes)
		frame.ImageDataBase64 = base64.StdEncoding.EncodeToString(imageData)
		pngBytes, err := encodeCTCPSimpleImagePNG(info, imageData)
		if err != nil {
			frame.PngError = err.Error()
		} else {
			frame.PngBase64 = base64.StdEncoding.EncodeToString(pngBytes)
		}
	default:
		return frame, fmt.Errorf("unsupported ipm image command: 0x%04X", uint32(head.NCmdId))
	}

	return frame, nil
}

func parseCTCPSpliceImagePayload(payload []byte) (cTCPSpliceImageInfoWire, []byte, []byte, error) {
	var info cTCPSpliceImageInfoWire
	if len(payload) < cTCPSpliceImageInfoWireSize {
		return info, nil, nil, fmt.Errorf("splice image payload too short: need %d, got %d", cTCPSpliceImageInfoWireSize, len(payload))
	}
	info.ImageDataLength = int32(binary.LittleEndian.Uint32(payload[0:4]))
	info.Width = int32(binary.LittleEndian.Uint32(payload[4:8]))
	info.Height = int32(binary.LittleEndian.Uint32(payload[8:12]))
	info.RouteID = int32(binary.LittleEndian.Uint32(payload[12:16]))
	offset := 16
	for index := 0; index < cTCPServerMaxCameraDirection; index++ {
		info.ChannelH[index] = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
		offset += 4
	}
	info.ValidCupNum = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	info.ColorMidLen = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	info.FlawArea = binary.LittleEndian.Uint32(payload[offset : offset+4])
	offset += 4
	info.FlawNum = binary.LittleEndian.Uint32(payload[offset : offset+4])
	offset += 4
	info.ColorCutY = math.Float32frombits(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	info.PixelBit = payload[offset]

	imageData, binData, err := splitCTCPImageBytes(payload, cTCPSpliceImageInfoWireSize, int(info.ImageDataLength), int(info.Width)*int(info.Height))
	return info, imageData, binData, err
}

func parseCTCPSimpleImagePayload(payload []byte) (cTCPSimpleImageInfoWire, []byte, error) {
	var info cTCPSimpleImageInfoWire
	if len(payload) < cTCPSimpleImageInfoWireSize {
		return info, nil, fmt.Errorf("image payload too short: need %d, got %d", cTCPSimpleImageInfoWireSize, len(payload))
	}
	info.ImageDataLength = int32(binary.LittleEndian.Uint32(payload[0:4]))
	info.Width = int32(binary.LittleEndian.Uint32(payload[4:8]))
	info.Height = int32(binary.LittleEndian.Uint32(payload[8:12]))
	info.RouteID = int32(binary.LittleEndian.Uint32(payload[12:16]))
	info.Verify = binary.LittleEndian.Uint32(payload[16:20])
	info.ImageFormat = payload[20]

	imageData, _, err := splitCTCPImageBytes(payload, cTCPSimpleImageInfoWireSize, int(info.ImageDataLength), 0)
	return info, imageData, err
}

func splitCTCPImageBytes(payload []byte, dataOffset int, imageLength int, binLength int) ([]byte, []byte, error) {
	if imageLength <= 0 {
		return nil, nil, fmt.Errorf("invalid image length: %d", imageLength)
	}
	if dataOffset+imageLength > len(payload) {
		return nil, nil, fmt.Errorf("image data truncated: offset=%d length=%d payload=%d", dataOffset, imageLength, len(payload))
	}
	imageData := append([]byte(nil), payload[dataOffset:dataOffset+imageLength]...)
	binOffset := dataOffset + imageLength
	if binLength <= 0 || binOffset+binLength > len(payload) {
		return imageData, nil, nil
	}
	return imageData, append([]byte(nil), payload[binOffset:binOffset+binLength]...), nil
}

func encodeCTCPSpliceImagePNG(info cTCPSpliceImageInfoWire, imageData []byte) ([]byte, error) {
	switch info.PixelBit {
	case cTCPSpliceImagePixelBitGray:
		return encodeCTCPGrayPNG(int(info.Width), int(info.Height), imageData)
	case cTCPSpliceImagePixelBitYUV422:
		return encodeCTCPYUV422PNG(int(info.Width), int(info.Height), imageData)
	default:
		if len(imageData) >= int(info.Width)*int(info.Height)*2 {
			return encodeCTCPYUV422PNG(int(info.Width), int(info.Height), imageData)
		}
		return encodeCTCPGrayPNG(int(info.Width), int(info.Height), imageData)
	}
}

func encodeCTCPSimpleImagePNG(info cTCPSimpleImageInfoWire, imageData []byte) ([]byte, error) {
	switch info.ImageFormat {
	case cTCPSimpleImageFormatMono:
		return encodeCTCPGrayPNG(int(info.Width), int(info.Height), imageData)
	case cTCPSimpleImageFormatYUV422:
		return encodeCTCPYUV422PNG(int(info.Width), int(info.Height), imageData)
	default:
		if len(imageData) >= int(info.Width)*int(info.Height)*2 {
			return encodeCTCPYUV422PNG(int(info.Width), int(info.Height), imageData)
		}
		return encodeCTCPGrayPNG(int(info.Width), int(info.Height), imageData)
	}
}

func encodeCTCPGrayPNG(width int, height int, imageData []byte) ([]byte, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image size: %dx%d", width, height)
	}
	pixelCount := width * height
	if len(imageData) < pixelCount {
		return nil, fmt.Errorf("gray image data too short: need %d, got %d", pixelCount, len(imageData))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for index := 0; index < pixelCount; index++ {
		y := index / width
		x := index - y*width
		value := imageData[index]
		img.SetRGBA(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
	}
	return encodeCTCPPNG(img)
}

func encodeCTCPYUV422PNG(width int, height int, imageData []byte) ([]byte, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image size: %dx%d", width, height)
	}
	pixelCount := width * height
	required := pixelCount * 2
	if len(imageData) < required {
		return nil, fmt.Errorf("yuv422 image data too short: need %d, got %d", required, len(imageData))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pairCount := pixelCount / 2
	for pairIndex := 0; pairIndex < pairCount; pairIndex++ {
		base := pairIndex * 4
		y0 := int(imageData[base])
		u := signedImageByte(imageData[base+1])
		y1 := int(imageData[base+2])
		v := signedImageByte(imageData[base+3])
		setCTCPImagePixel(img, width, pairIndex*2, y0, u, v)
		setCTCPImagePixel(img, width, pairIndex*2+1, y1, u, v)
	}
	if pixelCount%2 == 1 {
		last := pixelCount - 1
		value := imageData[last*2]
		img.SetRGBA(last%width, last/width, color.RGBA{R: value, G: value, B: value, A: 255})
	}
	return encodeCTCPPNG(img)
}

func setCTCPImagePixel(img *image.RGBA, width int, pixelIndex int, y int, u int, v int) {
	img.SetRGBA(pixelIndex%width, pixelIndex/width, color.RGBA{
		R: clampImageByte(float64(y) + 1.14*float64(v)),
		G: clampImageByte(float64(y) - 0.39*float64(u) - 0.58*float64(v)),
		B: clampImageByte(float64(y) + 2.03*float64(u)),
		A: 255,
	})
}

func signedImageByte(value byte) int {
	if value > 127 {
		return int(value) - 256
	}
	return int(value)
}

func clampImageByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	return uint8(value)
}

func encodeCTCPPNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bytesToHexPreview(data []byte, limit int) string {
	if len(data) == 0 || limit <= 0 {
		return ""
	}
	if len(data) < limit {
		limit = len(data)
	}
	return bytesToHex(data[:limit])
}

func bytesToHexTail(data []byte, limit int) string {
	if len(data) == 0 || limit <= 0 {
		return ""
	}
	start := len(data) - limit
	if start < 0 {
		start = 0
	}
	return bytesToHex(data[start:])
}

func bytesToHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(data)*3-1)
	for index, value := range data {
		if index > 0 {
			out = append(out, ' ')
		}
		out = append(out, hex[value>>4], hex[value&0x0F])
	}
	return string(out)
}

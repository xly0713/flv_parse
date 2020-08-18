package flv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/yutopp/go-amf0"
)

var headerSignature = []byte{0x46, 0x4c, 0x56} // F, L, V
const headerLength uint32 = 9

//Header flv header
type Header struct {
	Signature  string //FLV
	Version    uint8
	flags      uint8 //TypeFlagsReserved(5bit) + TypeFlagsAudio(1bit) + TypeFlagsReserved(1bit) + TypeFlagsVideo(1bit)
	dataOffSet uint32

	HasAudio bool
	HasVideo bool
}

//BodyInfo flv body info
type BodyInfo struct {
	audio               int64
	audioStartTimeStamp uint32
	audioEndTimeStamp   uint32
	video               int64
	videoStartTimeStamp uint32
	videoEndTimeStamp   uint32
}

//Parser flv parser struct
type Parser struct {
	reader   io.ReadSeeker
	header   *Header                   // flv header
	bodyInfo BodyInfo                  // flv body info
	metaInfo map[string]amf0.ECMAArray // flv body scriptData info
}

//NewFlvParser new flv parser instance
func NewFlvParser(r io.ReadSeeker) *Parser {
	return &Parser{
		reader: r,
	}
}

//ParseFlv parse flv file
func (fp *Parser) ParseFlv() error {
	if err := fp.parseHeader(); err != nil {
		return err
	}

	if err := fp.parseBody(); err != nil {
		return err
	}

	return nil
}

//Header return parsed flv header
func (fp *Parser) Header() *Header {
	return fp.header
}

//BodyInfo return parsed flv body info
func (fp *Parser) BodyInfo() *BodyInfo {
	return &fp.bodyInfo
}

//MetaInfo return parsed flv scriptData Info
func (fp *Parser) MetaInfo() map[string]amf0.ECMAArray {
	return fp.metaInfo
}

func (fp *Parser) parseHeader() error {
	bHeader := make([]byte, headerLength)
	_, err := io.ReadAtLeast(fp.reader, bHeader, int(headerLength))
	if err != nil {
		return err
	}

	header := &Header{}
	if err := parseHeader(bHeader, header); err != nil {
		return errors.Wrap(err, "parse flv header")
	}
	fp.header = header

	if header.dataOffSet > headerLength {
		offset := header.dataOffSet - headerLength
		if _, err := fp.reader.Seek(int64(offset), io.SeekCurrent); err != nil {
			return err
		}
	}

	return nil
}

func (fp *Parser) parseBody() error {
	bPreviousTagSize0 := make([]byte, 4) //always 0
	_, err := io.ReadAtLeast(fp.reader, bPreviousTagSize0, 4)
	if err != nil {
		return errors.Wrap(err, "failed to read PreviousTagSize0")
	}
	if !bytes.Equal(bPreviousTagSize0, []byte{0x00, 0x00, 0x00, 0x00}) {
		return errors.New("PreviousTagSize0 not 0")
	}

	if err := fp.parseTags(); err != nil {
		return errors.Wrap(err, "failed to parse tags")
	}

	return nil
}

func (fp *Parser) parseTags() error {
parseTagsLoop:
	for {
		bTagBuf := make([]byte, 1+3+3+1+3) // TagType(8) + DataSize(24) + TimeStamp(24) + TimeStampExtended(8) + streamID(24)
		_, err := io.ReadAtLeast(fp.reader, bTagBuf, 11)
		if err == io.EOF {
			break parseTagsLoop
		} else if err != nil {
			return err
		}

		tagType := ui8(bTagBuf[0:1]) & 0x1f
		dataSize := ui24(bTagBuf[1:4]) //FIXME: check dataSize positive

		dataBuf := make([]byte, dataSize)
		if _, err := io.ReadAtLeast(fp.reader, dataBuf, int(dataSize)); err != nil {
			return err
		}

		bPreviousTagSizeN := make([]byte, 4)
		if _, err := io.ReadAtLeast(fp.reader, bPreviousTagSizeN, 4); err != nil {
			return err
		}

		previousTagSizeN := binary.BigEndian.Uint32(bPreviousTagSizeN)
		if previousTagSizeN != 11+dataSize {
			return errors.New("previousTagSize value incorrect")
		}

		timeStamp := ui24(bTagBuf[4:7])
		timeStampExtended := ui8(bTagBuf[7:8])
		tagTimeStamp := (uint32(timeStampExtended) << 24) + timeStamp

		switch tagType {
		case 0x08: //audio
			fp.bodyInfo.audio++
			if fp.bodyInfo.audio == 1 {
				fp.bodyInfo.audioStartTimeStamp = tagTimeStamp
			}
			fp.bodyInfo.audioEndTimeStamp = tagTimeStamp

			//TODO: parse audio data

		case 0x09: //video
			fp.bodyInfo.video++
			if fp.bodyInfo.video == 1 {
				fp.bodyInfo.videoStartTimeStamp = tagTimeStamp
			}
			fp.bodyInfo.videoEndTimeStamp = tagTimeStamp

			//TODO: parse video data
			fp.parseVideoTag(dataBuf)

		case 0x12: //scriptData
			fp.metaInfo = make(map[string]amf0.ECMAArray)

			if err := decodeScriptData(bytes.NewReader(dataBuf), fp.metaInfo); err != nil {
				return err
			}
		default:
			return errors.New("unknown flv tag type")
		}
	}

	return nil
}

func (fp *Parser) parseVideoTag(dataBuf []byte) error {
	fc := ui8(dataBuf[0:1])
	frameType := (fc >> 4) //1:keyframe  2:inter frame  3:disposable inter frame(H.263 only)  4:generated keyframe(server use only)  5:video info/command frame
	codecID := fc & 0x0f   //1:JPEG  2:Sorenson H.263  3.Screen video  4.On2 VP6  5:On2 VP6 with alpha channel  6:Screen video version 2  7:AVC

	if frameType == 5 { //video info/command frame
		seek := ui8(dataBuf[1:2])
		if seek == 0 {
			fmt.Println("Start of client-side seeking video frame squeunce")
		} else if seek == 1 {
			fmt.Println("End of client-side seeking video frame squeunce")
		}
		return nil
	}

	switch codecID {
	case 2:
		//H263VIDEOPACKET
	case 3:
		//SCREENVIDEOPACKET
	case 4:
		//VP6FLVVIDEOPACKET
	case 5:
		//VP6FLVALPHAVIDEOPACKET
	case 6:
		//SCREENV2VIDEOPACKET
	case 7:
		//AVCVIDEOPACKET
		return fp.avcVideoPacket(dataBuf, frameType)
	default:
	}

	//fmt.Printf("FrameType: %d, CodecID: %d\n", frameType, codecID)

	return nil
}

func (fp *Parser) avcVideoPacket(dataBuf []byte, frameType uint8) error {
	videoData := dataBuf[1:]
	if len(videoData) <= 4 {
		return errors.New("invalid AVCVIDEOPACKET")
	}

	// 0:AVC sequence header  1:AVC NALU   2: AVC end of sequence
	avcPacketType := ui8(videoData[0:1])
	// Composition time offset
	compositionTime := ui24(videoData[1:4])
	// h.264 raw data  (1) AVCDecoderConfigurationRecord while AVCPacketType is 0  (2) one or more NALUs while AVCPacketType is 1  (3) otherwise empty
	dataN := dataBuf[4:]

	if avcPacketType != 1 && compositionTime != 0 {
		return errors.New("Composition time offset not while AVCPacketType not 1(AVC NALU)")
	}

	fmt.Printf("[%-10d], FrameType: %d, CodecID: 7, AVCPacketType: %d, CompositionTime: %-5d, VideoDataLen: %d\n",
		fp.bodyInfo.video, frameType, avcPacketType, compositionTime, len(dataN))

	return nil
}

func parseHeader(bHeader []byte, header *Header) error {
	if !bytes.Equal(bHeader[0:3], headerSignature) {
		return errors.New("invalid flv signature")
	}
	header.Signature = string(headerSignature)

	header.Version = ui8(bHeader[3:4])

	header.flags = ui8(bHeader[4:5])
	if header.flags&0xf8 != 0 {
		return errors.New("previous 5 bit not 0")
	}
	if header.flags&0x04 == 0x04 { // the 6th bit
		header.HasAudio = true
	}
	if header.flags&0x02 != 0 {
		return errors.New("the 7th bit not 0")
	}
	if header.flags&0x01 == 1 { // the 8th bit
		header.HasVideo = true
	}

	header.dataOffSet = binary.BigEndian.Uint32(bHeader[5:9])

	return nil
}

func decodeScriptData(r io.Reader, scriptDataMap map[string]amf0.ECMAArray) error {
	dec := amf0.NewDecoder(r)

scriptTagParse:
	for {
		var key string
		if err := dec.Decode(&key); err != nil {
			if err == io.EOF {
				break scriptTagParse
			}

			return errors.Wrap(err, "failed to decode scriptData key")
		}

		var value amf0.ECMAArray
		if err := dec.Decode(&value); err != nil {
			return errors.Wrap(err, "failed to decode scriptData map Value")
		}

		scriptDataMap[key] = value
	}

	return nil
}

func ui8(b []byte) uint8 {
	_ = b[0]
	return uint8(b[0])
}

func ui24(b []byte) uint32 {
	_ = b[2]
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

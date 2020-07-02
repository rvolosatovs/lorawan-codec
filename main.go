package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/mohae/deepcopy"
	"go.thethings.network/lorawan-stack/v3/pkg/band"
	"go.thethings.network/lorawan-stack/v3/pkg/crypto"
	"go.thethings.network/lorawan-stack/v3/pkg/encoding/lorawan"
	"go.thethings.network/lorawan-stack/v3/pkg/jsonpb"
	"go.thethings.network/lorawan-stack/v3/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/v3/pkg/types"
)

var json = &jsonpb.GoGoJSONPb{
	EmitDefaults: true,
	OrigName:     true,
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
}

func macBuffer(pld *ttnpb.MACPayload) []byte {
	if pld.FPort == 0 && len(pld.FRMPayload) > 0 {
		return pld.FRMPayload
	}
	return pld.FOpts
}

type Config struct {
	PHY        band.Band
	MACVersion ttnpb.MACVersion

	AppKey     types.AES128Key
	NwkSEncKey types.AES128Key
}

func decode(w io.Writer, r io.Reader, conf Config) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	var msg ttnpb.Message
	if err := lorawan.UnmarshalMessage(b, &msg); err != nil {
		return fmt.Errorf("failed to decode frame as LoRaWAN: %w", err)
	}
	pld, mic := func() (interface{}, []byte) {
		switch msg.MHDR.MType {
		case ttnpb.MType_JOIN_REQUEST:
			return msg.GetJoinRequestPayload(), msg.MIC

		case ttnpb.MType_REJOIN_REQUEST:
			return msg.GetRejoinRequestPayload(), msg.MIC

		case ttnpb.MType_JOIN_ACCEPT:
			pld := msg.GetJoinAcceptPayload()
			pldBuf, err := crypto.DecryptJoinAccept(conf.AppKey, pld.Encrypted)
			if err != nil {
				log.Printf("Failed to decrypt JoinAccept: %s", err)
				return pld, nil
			}
			n := len(pldBuf)
			if n < 4 {
				log.Printf("Invalid JoinAccept length, expected at least 4 bytes, got: %d", n)
				return pld, nil
			}
			pldBuf, mic := pldBuf[:n-4], pldBuf[n-4:]
			decPld := deepcopy.Copy(pld).(*ttnpb.JoinAcceptPayload)
			if err := lorawan.UnmarshalJoinAcceptPayload(pldBuf, decPld); err != nil {
				log.Printf("Failed to decode JoinAccept: %s", err)
				return pld, mic
			}
			return decPld, mic

		case ttnpb.MType_UNCONFIRMED_DOWN, ttnpb.MType_CONFIRMED_DOWN:
			pld := msg.GetMACPayload()
			macBuf := macBuffer(pld)
			if len(macBuf) > 0 {
				log.Println("NOTE: Downlink MAC command parsing is not implemented yet")
			}
			if pld.FPort > 0 {
				log.Println("NOTE: Downlink application payload decryption is not implemented yet")
			}
			return pld, msg.MIC

		case ttnpb.MType_UNCONFIRMED_UP, ttnpb.MType_CONFIRMED_UP:
			pld := msg.GetMACPayload()
			macBuf := macBuffer(pld)
			if len(macBuf) > 0 && (len(pld.FOpts) == 0 || conf.MACVersion.EncryptFOpts()) {
				for msb := uint32(0); msb < 0xff; msb++ {
					fCnt := msb<<8 | pld.FCnt
					macBuf, err := crypto.DecryptUplink(conf.NwkSEncKey, pld.DevAddr, fCnt, macBuf, pld.FPort != 0)
					if err != nil {
						log.Printf("Failed to decrypt MAC buffer with FCnt %v: %s", fCnt, err)
					} else if pld.FPort == 0 {
						pld.FRMPayload = macBuf
						break
					} else {
						pld.FOpts = macBuf
						break
					}
				}
			}
			var macCommands []*ttnpb.MACCommand
			for r := bytes.NewReader(macBuf); r.Len() > 0; {
				cmd := &ttnpb.MACCommand{}
				if err := lorawan.DefaultMACCommands.ReadUplink(conf.PHY, r, cmd); err != nil {
					log.Printf("Failed to read MAC command: %s", err)
					break
				}
				macCommands = append(macCommands, cmd)
			}
			if pld.FPort > 0 {
				log.Printf("NOTE: Uplink application payload decryption is not implemented yet")
			}
			return struct {
				*ttnpb.MACPayload
				MACCommands []*ttnpb.MACCommand `json:"mac_commands,omitempty"`
			}{
				MACPayload:  pld,
				MACCommands: macCommands,
			}, msg.MIC
		default:
			log.Printf("Unmatched FType: %v", msg.MHDR.MType)
			return nil, msg.MIC
		}
	}()

	type mhdr struct {
		MType ttnpb.MType `json:"m_type"`
		Major ttnpb.Major `json:"major"`
	}
	if err := json.NewEncoder(w).Encode(struct {
		MHDR    mhdr        `json:"mhdr"`
		MIC     []byte      `json:"mic"`
		Payload interface{} `json:"payload"`
	}{
		MHDR: mhdr{
			MType: msg.MHDR.MType,
			Major: msg.MHDR.Major,
		},
		MIC:     mic,
		Payload: pld,
	}); err != nil {
		return fmt.Errorf("failed to encode frame to JSON: %w", err)
	}
	return nil
}

func main() {
	quiet := flag.Bool("quiet", false, "Whether to suppress all log output")

	encode := flag.Bool("encode", false, "Whether encoding should be performed")

	bandID := flag.String("band", "EU_863_870", "Band name")

	macStr := flag.String("mac", "1.0.4", "MAC version")
	phyStr := flag.String("phy", "1.0.3-a", "PHY version")

	appKeyStr := flag.String("app_key", "", "AppKey")

	appSKeyStr := flag.String("app_s_key", "", "AppSKey")
	fNwkSIntKeyStr := flag.String("f_nwk_s_int_key", "", "FNwkSIntKey")
	nwkSEncKeyStr := flag.String("nwk_s_enc_key", "", "NwkSEncKey")
	sNwkSIntKeyStr := flag.String("s_nwk_s_int_key", "", "SNwkSIntKey")

	useBase64 := flag.Bool("base64", false, "Use base64 encoding for LoRaWAN frames")
	useHex := flag.Bool("hex", false, "Use hex encoding for LoRaWAN frames")

	flag.Parse()

	if *useBase64 && *useHex {
		log.Fatal("Only one of `base64` or `hex` can be specified")
	}

	if *quiet {
		log.SetOutput(ioutil.Discard)
	}

	var macVersion ttnpb.MACVersion
	if err := macVersion.UnmarshalText([]byte(*macStr)); err != nil {
		log.Fatalf("Failed to parse MAC version: %s", err)
	}
	var phyVersion ttnpb.PHYVersion
	if err := phyVersion.UnmarshalText([]byte(*phyStr)); err != nil {
		log.Fatalf("Failed to parse PHY version: %s", err)
	}

	phy, err := band.GetByID(*bandID)
	if err != nil {
		log.Fatalf("Failed to query band by ID: %s", err)
	}
	phy, err = phy.Version(phyVersion)
	if err != nil {
		log.Fatalf("Failed to query band version: %s", err)
	}

	var appKey types.AES128Key
	if *appKeyStr != "" {
		if err := appKey.UnmarshalText([]byte(*appKeyStr)); err != nil {
			log.Fatalf("Failed to parse AppKey: %s", err)
		}
	}

	var appSKey types.AES128Key
	if *appSKeyStr != "" {
		if err := appSKey.UnmarshalText([]byte(*appSKeyStr)); err != nil {
			log.Fatalf("Failed to parse AppSKey: %s", err)
		}
	}
	var sNwkSIntKey types.AES128Key
	if *sNwkSIntKeyStr != "" {
		if macVersion.Compare(ttnpb.MAC_V1_1) < 0 {
			log.Fatalf("SNwkSIntKey must not be specified for MAC version %s", *macStr)
		}
		if err := sNwkSIntKey.UnmarshalText([]byte(*sNwkSIntKeyStr)); err != nil {
			log.Fatalf("Failed to parse SNwkSIntKey: %s", err)
		}
	}
	var nwkSEncKey types.AES128Key
	if *nwkSEncKeyStr != "" {
		if macVersion.Compare(ttnpb.MAC_V1_1) < 0 {
			log.Fatalf("NwkSEncKey must not be specified for MAC version %s", *macStr)
		}
		if err := nwkSEncKey.UnmarshalText([]byte(*nwkSEncKeyStr)); err != nil {
			log.Fatalf("Failed to parse NwkSEncKey: %s", err)
		}
	}
	var fNwkSIntKey types.AES128Key
	if *fNwkSIntKeyStr != "" {
		if err := fNwkSIntKey.UnmarshalText([]byte(*fNwkSIntKeyStr)); err != nil {
			log.Fatalf("Failed to parse FNwkSIntKey: %s", err)
		}
		if macVersion.Compare(ttnpb.MAC_V1_1) < 0 {
			sNwkSIntKey = fNwkSIntKey
			nwkSEncKey = fNwkSIntKey
		}
	}
	conf := Config{
		PHY:        phy,
		MACVersion: macVersion,
		AppKey:     appKey,
		NwkSEncKey: nwkSEncKey,
	}

	if *encode {
		log.Fatal("Encoding not implemented")
	}

	byteReader := &bytes.Reader{}
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		byteReader.Reset(sc.Bytes())
		var r io.Reader
		switch {
		case *useBase64:
			r = base64.NewDecoder(base64.StdEncoding, byteReader)
		case *useHex:
			r = hex.NewDecoder(byteReader)
		default:
			r = byteReader
		}
		if err := decode(os.Stdout, r, conf); err != nil {
			log.Printf("Failed to decode frame: %s", err)
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("Failed to read stdin: %s", err)
	}
}

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/rvolosatovs/lorawan-stack/v3/pkg/band"
	"github.com/rvolosatovs/lorawan-stack/v3/pkg/crypto"
	"github.com/rvolosatovs/lorawan-stack/v3/pkg/encoding/lorawan"
	"github.com/rvolosatovs/lorawan-stack/v3/pkg/jsonpb"
	"github.com/rvolosatovs/lorawan-stack/v3/pkg/ttnpb"
	"github.com/rvolosatovs/lorawan-stack/v3/pkg/types"
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

func main() {
	encode := flag.Bool("encode", false, "Whether encoding should be performed")

	bandID := flag.String("band", "EU_863_870", "Band name")

	macStr := flag.String("mac", "1.0.4", "MAC version")
	phyStr := flag.String("phy", "1.0.3-a", "PHY version")

	appSKeyStr := flag.String("app_s_key", "", "AppSKey")
	fNwkSIntKeyStr := flag.String("f_nwk_s_int_key", "", "FNwkSIntKey")
	nwkSEncKeyStr := flag.String("nwk_s_enc_key", "", "NwkSEncKey")
	sNwkSIntKeyStr := flag.String("s_nwk_s_int_key", "", "SNwkSIntKey")

	flag.Parse()

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

	b, err := ioutil.ReadAll(hex.NewDecoder(os.Stdin))
	if err != nil {
		log.Fatalf("Failed to read stdin: %s", err)
	}

	if *encode {
		log.Fatal("Encoding not implemented")
	}
	var msg ttnpb.Message
	if err := lorawan.UnmarshalMessage(b, &msg); err != nil {
		log.Fatalf("Failed to decode frame: %s", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		MHDR    ttnpb.MHDR  `json:"mhdr"`
		MIC     []byte      `json:"mic"`
		Payload interface{} `json:"payload"`
	}{
		MHDR: msg.MHDR,
		MIC:  msg.MIC,
		Payload: func() interface{} {
			type macPayload struct {
				*ttnpb.MACPayload
				MACCommands []*ttnpb.MACCommand `json:"mac_commands,omitempty"`
			}

			switch msg.MHDR.MType {
			case ttnpb.MType_JOIN_REQUEST:
				return msg.GetJoinRequestPayload()

			case ttnpb.MType_REJOIN_REQUEST:
				return msg.GetRejoinRequestPayload()

			case ttnpb.MType_JOIN_ACCEPT:
				return msg.GetJoinAcceptPayload()

			case ttnpb.MType_UNCONFIRMED_DOWN, ttnpb.MType_CONFIRMED_DOWN:
				pld := msg.GetMACPayload()
				macBuf := macBuffer(pld)
				if len(macBuf) > 0 {
					log.Printf("NOTE: Downlink MAC command parsing is not implemented yet")
				}
				if pld.FPort > 0 {
					log.Printf("NOTE: Downlink application payload decryption is not implemented yet")
				}
				return pld

			case ttnpb.MType_UNCONFIRMED_UP, ttnpb.MType_CONFIRMED_UP:
				pld := msg.GetMACPayload()
				macBuf := macBuffer(pld)
				if len(macBuf) > 0 && (len(pld.FOpts) == 0 || macVersion.EncryptFOpts()) {
					for msb := uint32(0); msb < 0xff; msb++ {
						fCnt := msb<<8 | pld.FCnt
						macBuf, err = crypto.DecryptUplink(nwkSEncKey, pld.DevAddr, fCnt, macBuf, pld.FPort != 0)
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
					if err := lorawan.DefaultMACCommands.ReadUplink(phy, r, cmd); err != nil {
						log.Printf("Failed to read MAC command: %s", err)
						break
					}
					macCommands = append(macCommands, cmd)
				}
				if pld.FPort > 0 {
					log.Printf("NOTE: Uplink application payload decryption is not implemented yet")
				}
				return macPayload{
					MACPayload:  pld,
					MACCommands: macCommands,
				}
			default:
				log.Printf("Unmatched FType: %v", msg.MHDR.MType)
				return nil
			}
		}(),
	}); err != nil {
		log.Fatalf("Failed to write JSON frame to stdout: %s", err)
	}
}

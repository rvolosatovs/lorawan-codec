package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"go.thethings.network/lorawan-stack/pkg/encoding/lorawan"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/types"
)

func init() {
	log.SetFlags(0)
}

func main() {
	encode := flag.Bool("encode", false, "Whether encoding should be performed")

	macStr := flag.String("mac", "1.0.4", "MAC version")
	phyStr := flag.String("phy", "1.0.3-b", "PHY version")

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

	var appSKey types.AES128Key
	if *appSKeyStr != "" {
		if err := appSKey.UnmarshalText([]byte(*appSKeyStr)); err != nil {
			log.Fatalf("Failed to parse AppSKey: %s", err)
		}
	}
	var fNwkSIntKey types.AES128Key
	if *fNwkSIntKeyStr != "" {
		if err := fNwkSIntKey.UnmarshalText([]byte(*fNwkSIntKeyStr)); err != nil {
			log.Fatalf("Failed to parse FNwkSIntKey: %s", err)
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

	b, err := ioutil.ReadAll(os.Stdout)
	if err != nil {
		log.Fatalf("Failed to read stdout: %s", err)
	}

	if *encode {
		log.Fatal("Encoding not implemented")
	}
	var msg ttnpb.Message
	if err := lorawan.UnmarshalMessage(b, &msg); err != nil {
		log.Fatalf("Failed to decode frame: %s", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(msg); err != nil {
		log.Fatalf("Failed to write JSON frame to stdout: %s", err)
	}
}

# Description

`lorawan-codec` encodes and decodes LoRaWAN frames.

# Examples

## Simple hex-encoded frame decoding

```bash
$ printf '40BF000027850100030706FF08FEB3E9A6' | go run github.com/rvolosatovs/lorawan-codec -hex | jq
{
  "mhdr": {
    "m_type": "UNCONFIRMED_UP"
  },
  "mic": "/rPppg==",
  "payload": {
    "f_hdr": {
      "dev_addr": "270000BF",
      "f_ctrl": {
        "adr": true
      },
      "f_cnt": 1,
      "f_opts": "AwcG/wg="
    },
    "mac_commands": [
      {
        "cid": "CID_LINK_ADR",
        "Payload": {
          "link_adr_ans": {
            "channel_mask_ack": true,
            "data_rate_index_ack": true,
            "tx_power_index_ack": true
          }
        }
      },
      {
        "cid": "CID_DEV_STATUS",
        "Payload": {
          "dev_status_ans": {
            "battery": 255,
            "margin": 8
          }
        }
      }
    ]
  }
}
```

## Decode and decrypt frames from gateway log stream on the fly

```bash
$ tail -f ~/mnt/gateway/var/log/pkt_fwd.log | rg -o --line-buffered 'JSON (down|up): (.*)' -r '$2' | jq --unbuffered -r '(select(.rxpk? != null) | .rxpk | .[] | .data), (select(.txpk? != null) | .txpk.data)' | go run main.go -base64 -app_key 01020304050607080102030405060708 | jq
{
  "mhdr": {
    "m_type": "JOIN_REQUEST",
    "major": "LORAWAN_R1"
  },
  "mic": "Vv4FEg==",
  "payload": {
    "join_eui": "01020304DEADBEEF",
    "dev_eui": "DEADBEEF01020304",
    "dev_nonce": "F8EC"
  }
}
{
  "mhdr": {
    "m_type": "JOIN_ACCEPT",
    "major": "LORAWAN_R1"
  },
  "mic": "GL26bA==",
  "payload": {
    "encrypted": "09jOfnDN/QdEaLF8zMUdM6m34QqmpSOl+S+1FIXEVcQ=",
    "join_nonce": "00001A",
    "net_id": "000000",
    "dev_addr": "013BBB8B",
    "dl_settings": {},
    "rx_delay": 5,
    "cf_list": {
      "freq": [
        8671000,
        8673000,
        8675000,
        8677000,
        8679000
      ]
    }
  }
}
```

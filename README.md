# Description

`lorawan-codec` encodes and decodes LoRaWAN frames.

# Example

Simple one-off hex-encoded frame decoding:

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

Decode frames from gateway log stream:
```bash
    $ tail -f ~/mnt/gateway/var/log/pkt_fwd.log | rg -o --line-buffered 'JSON (down|up): (.*)' -r '$2' | jq --unbuffered -r '.[] | .[] | .data?' | go run github.com/rvolosatovs/lorawan-codec -base64 -f_nwk_s_int_key 88A4CB739A3579D7BB227156FBEDC227 | jq 
    {
      "mhdr": {},
      "mic": "9KSzhQ==",
      "payload": {
        "join_eui": "01020304DEADBEEF",
        "dev_eui": "DEADBEEF01020304",
        "dev_nonce": "F17C"
      }
    }
    {
      "mhdr": {
        "m_type": "UNCONFIRMED_UP"
      },
      "mic": "WRV9FA==",
      "payload": {
        "f_hdr": {
          "dev_addr": "2700002A",
          "f_ctrl": {
            "adr": true
          }
        }
      }
    }
```

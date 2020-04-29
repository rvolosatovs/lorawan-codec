# Description

`lorawan-codec` encodes and decodes LoRaWAN frames.

# Example

```bash
    $ printf '40BF000027850100030706FF08FEB3E9A6' | go run github.com/rvolosatovs/lorawan-codec | jq
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

# Host burner

Golang app intended for burning/reading data from host (and not from Armory)

## Compilation 

In `host-burner/` directory run:

```
go build burner.go
```

This will create `burner` binary.

## Usage

```
burner [-burn] [-binary] -type TYPE -file FILE -device DEVICE
```

+ `-burn` If provided, data will be burned from local file to μSD. Otherwise, data will be loaded from μSD. Optional argument.
+ `-binary` When burning, data will be encoded in base64 before writing to block device. When loading data from device, data will be decoded from base64 before writing to a local file. This useful when you want to burn `.psbt` file exported from the Electrum, or when you want to load signed transaction to the Electrum. Optional argument. 
+ `-type` Specifies type of data that is transfered. Can be `mnemonic`, `password`, `psbt` or `tx`. Mandatory argument.
+ `-file` Specifies the local file. Mandatory argument.
+ `-device` Specifies the block device. Mandatory argument.


Note that we want to use device name (say `/dev/sdb`) and not partition name (say `/dev/sdb1`). You can identify devices via `df` command.

You probably want to run app via sudo (otherwise OS system can block you form reading/writing to the device)

## Warnings

This app can expose your mnemonics and password. This app can overwrite your mnemonics and password or anything else stored on the targeted block device. 
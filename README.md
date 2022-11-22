# Blind signer

The USB armory can function as a secure blind signer for deep cold, high value bitcoin transactions.

## Single sig signing scenario
* A password-protected seed (BIP39) is stored on a microSD card labelled **S**, in a secure location. The encrypting password is not present at the location.
* The password is stored on a separate microSD card labelled **P**.
* The owner uses electrum or some other wallet to generate a transaction with a watch-only wallet and saves it to a microSD card labelled **T**.
* The owner goes to the secure location, bringing: 
  * the USB armory, powered by a powerbank
  * The microSDs **P** and **T**
* At the secure location, the owner plugs the USB armory into the powerbank; it starts booting (blinking white LED); and owner waits for it to become ready (solid white LED)
* The owner inserts the **S** card; when the seed is copied to RAM the blue LED turns on
* The owner takes out the **S** card (blue LED turns off), and inserts the **P** card; when the password is copied into RAM the blue LED turns on
* The owner takes out the **P** card (blue LED turns off), and inserts the **T** card;
* The device decrypts the seed with the password, derives the necessary private keys for the transaction and signs the transaction
* The device saves the signed transaction to **T**
* The device clears all sensitive info from memory
* The blue LED turns solid and the white LED starts blinking, indicating final success
* Owner removes **T** and disconnects the USB armory from the powerbank
* Back home, Owner loads the signed transaction from **T** and broadcasts
* If any run-time error occurs, the armory stops executing the protocol, and the blue LED starts blinking

## Building

Build the [TamaGo compiler](https://github.com/usbarmory/tamago-go)
(or use the [latest binary release](https://github.com/usbarmory/tamago-go/releases/latest)):

```bash
wget https://github.com/usbarmory/tamago-go/archive/refs/tags/latest.zip
unzip latest.zip
cd tamago-go-latest/src && ./all.bash
cd ../bin && export TAMAGO=`pwd`/go
```

Build the application executables as follows:

```bash
cd burner
make imx TARGET=usbarmory
```

```bash
cd signer
make imx TARGET=usbarmory
```

## Flashing

We can run the apps trough serial download protocol on an armory connected to the host, and also flash them on either an external microSD card, connected to the host with a built-in or externally connected card reader, or the internal eMMC. You can find instrunctions for both these options [here](https://github.com/usbarmory/usbarmory/wiki/Boot-Modes-(Mk-II))

## Running

### Burner

To run the burner app, you first need to configure the network:

On Linux:

```bash
ip link set usb0 up
ip addr add 10.0.0.2/24 dev usb0
iptables -t nat -A POSTROUTING -s 10.0.0.1/32 -o wlan0 -j MASQUERADE
```

The interface names usb0 and wlan0 should be adjusted to reflect your configuration.

For more info, or other OS support, check the [wiki](https://github.com/usbarmory/usbarmory/wiki/Host-communication).

After succesfully configure the network, you can ssh into it:

```bash
ssh root@10.0.0.1
```

The SSH server exposes a console with the following commands:

```
exit, quit                                            # close session
help                                                  # this help
info                                                  # device information
led             (white|blue) (on|off)                 # LED control
read            (mnemonic|password|psbt|tx)           # read from SD Card
reboot                                                # reset device
write           (mnemonic|password|psbt) (.+)         # write to SD Card
```

### Signer

To run the signer, you just need to plug the armory to a power source.

## TO-DO

### Multi-sig
The software should support multi-sig operations as well, in case the owner requires multiple secure locations.

See [Bitcoin Secure Multisig Setup (BSMS)](https://bips.xyz/129).

### Overwriting sensitive data
In order to try and guard against cold boot attacks and data remanence, we can attempt to overwrite sensitive RAM areas, like private key storage. The private key type in btcd seems to be based on this library, which allows zeroing of the key:

https://github.com/decred/dcrd/blob/master/dcrec/secp256k1/privkey.go#L76

We could attempt to do even more, setting it to a series of zebra stripes (1010, 0101). We check what other libraries like Sodium do. From a quick check, Sodium just overwrites with zeros, so let's start with that.

### Additional security
* The USB armory could be fused to only accept transactions signed by a public key

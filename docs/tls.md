# TLS / certificate setup

panoptes needs to decrypt HTTPS traffic to inspect it. it does this by generating its own root CA certificate and dynamically signing per-host leaf certificates at runtime.

this is the same technique every intercepting proxy uses — burp, charles, mitmproxy, fiddler, all of them.

## how it works

1. on first run, panoptes generates a **2048-bit RSA self-signed root CA** and saves it to `certs/`:
   - `certs/panoptes-ca.crt` — the public certificate (install this in your browser/OS)
   - `certs/panoptes-ca.key` — the private key (keep this safe, never share it)

2. when a browser connects to an HTTPS site through panoptes, the proxy:
   - sees the hostname from the TLS SNI (Server Name Indication)
   - dynamically generates a **leaf certificate** for that exact hostname
   - signs it with the root CA
   - presents it to the browser

3. since your browser trusts the root CA, it trusts the leaf certificate → no TLS warnings

## install the CA

### linux (system-wide)

```sh
sudo cp certs/panoptes-ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

### macOS (system-wide)

```sh
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain certs/panoptes-ca.crt
```

### firefox

firefox uses its own certificate store, not the system one.

1. open firefox → Preferences → Privacy & Security
2. scroll to **Certificates** → click **View Certificates**
3. go to the **Authorities** tab → click **Import**
4. select `certs/panoptes-ca.crt`
5. check **"Trust this CA to identify web sites"**
6. click OK

### chrome / chromium

chrome uses the system certificate store on all platforms. install it with the OS method above, then restart chrome.

### windows

why in hell would you use windows for hacking (or anything else). anyway here's the way to add the certificate in windows suggested by ai as I don't use windows (thank god for this).

```powershell
certutil -addstore Root certs\panoptes-ca.crt
```

## where certs live

```
panoptes/
├── certs/
│   ├── panoptes-ca.crt    ← Root CA certificate (public)
│   └── panoptes-ca.key    ← Root CA private key (secret)
```

these are gitignored. don't commit them.

if you delete the `certs/` directory, panoptes will generate a fresh CA on next run. you'll need to reinstall it in your browser.

## troubleshooting

### "this site is not secure" / `SEC_ERROR_UNKNOWN_ISSUER`

the CA isn't trusted. make sure you installed `panoptes-ca.crt` in the right store for your browser.

### `PR_END_OF_FILE_ERROR`

the response body transfer-encoding got mangled. this is a known issue with certain chunked responses — restart panoptes and try again.

### "proxy is refusing connections"

panoptes isn't running. make sure it's still alive in your terminal. check for port conflicts on 8080.

### certificate expired

leaf certificates expire after 24 hours. this is normal — panoptes generates fresh ones on the fly. if you see this, the clock on your machine might be wrong.

### ca cert expiring

the root CA is valid for 10 years. if it's somehow expired, delete `certs/` and restart panoptes to regenerate it.

## security notes

- the root CA private key (`panoptes-ca.key`) can decrypt any HTTPS traffic you intercept through panoptes. **never share it.**
- only install the CA in browsers and machines you control.
- stop panoptes and remove the CA from your browser's trust store when you're done.

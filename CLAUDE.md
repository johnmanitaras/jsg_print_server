# Print Server Project

## ESC/POS Emulator

For development and testing, we use **escpos-netprinter** - a Docker-based ESC/POS emulator that receives real print data from the Go print server and renders receipts as HTML via PHP.

**This is the only emulator.** Do not add alternative emulators, Python-based receivers, React-based simulators, or third-party preview tools without discussing first.

### How it works

1. The Docker container (`escpos-emulator`) listens on **TCP port 9100** (same port as a real thermal printer)
2. The Go print server (`printserver.exe`) sends print jobs to `127.0.0.1:9100`
3. The emulator parses ESC/POS commands using PHP (`esc2html.php`) and renders receipts as HTML
4. View rendered receipts at **http://localhost:8888**

### Running the emulator

```
docker start escpos-emulator
```

If the container doesn't exist yet:
```
docker run -d --name escpos-emulator -p 9100:9100 -p 8888:80 gilbertfl/escpos-netprinter:3.2
```

- Print receiver: `TCP 9100`
- Web preview: `http://localhost:8888`
- Requires Docker Desktop to be running

### Printer configuration

In the print server web UI or `config.yaml`, configure a printer pointing to `127.0.0.1:9100` (type: `network`). Print jobs sent to that printer will be captured by the emulator.

### Local customisations

The `escpos-netprinter/` folder contains a local clone with the `esc2html_fixed.css` font fix. The Docker container runs from the `gilbertfl/escpos-netprinter:3.2` image. Do not create alternative emulator scripts.

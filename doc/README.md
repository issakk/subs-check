# BestSub Usage Guide

> ⚠️ **Important Notice**  
> This project is under active development.  
> Configuration files may change frequently.  
> Please pay close attention to the documentation updates.

[中文文档](./README_zh.md) | English Documentation

## Direct Execution

1. Select the appropriate file from the [releases](https://github.com/bestruirui/BestSub/releases) based on your system
2. Download [config.example.yaml](https://raw.githubusercontent.com/bestruirui/BestSub/master/doc/config.example.yaml) and [rename.yaml](https://raw.githubusercontent.com/bestruirui/BestSub/master/doc/rename.yaml) to the `config` folder
3. Refer to the [Configuration Documentation](./config.md) to modify the configuration file, then rename it to `config.yaml`
4. Run the application

## Docker

```bash
mkdir -p /path/to/config
```

```bash
docker run -itd \
    --name bestsub \
    -p 8080:8080 \
    -v /path/to/config:/app/config \
    -v /path/to/output:/app/output \
    --restart=always \
    ghcr.io/bestruirui/bestsub
```

## Run from Source Code

```bash
go run main.go -f /path/to/config.yaml -r /path/to/rename.yaml
```

## Custom Speed Test URL

> (Optional) Since some nodes block common speed test URLs, you may need to create your own speed test URL

- Deploy the [worker](./cloudflare/worker.js) to Cloudflare Workers

- Set `speed-test-url` to your worker URL

```yaml
speed-test-url: https://your-worker-url/speedtest?bytes=1000000
```

## Save Method Configuration

- 📁 Local Save: Save results locally, default location is the output folder in the executable directory
- ☁️ R2: Save results to Cloudflare R2 bucket [Configuration Guide](./r2.md)
- 💾 Gist: Save results to GitHub Gist [Configuration Guide](./gist.md)
- 🌐 WebDAV: Save results to WebDAV server [Configuration Guide](./webdav.md)

## Subscription Usage

Recommended to run directly in tun mode

My own Windows application for direct execution: [minihomo](https://github.com/bestruirui/minihomo)

- Download [base.yaml](./doc/base.yaml)
- Replace the corresponding links in the file with your own

Example:

```yaml
proxy-providers:
  ProviderALL:
    url: https:// # Replace this with your own link
    type: http
    interval: 600
    proxy: DIRECT
    health-check:
      enable: true
      url: http://www.google.com/generate_204
      interval: 60
    path: ./proxy_provider/ALL.yaml
```

If using `local` save method:

```yaml
proxy-providers:
  ProviderALL:
    file: /path/to/all.yaml
    type: file
```

## Automatic Subscription Update

Implement automatic subscription update after detection

Refer to the `mihomo` option in the [Configuration Documentation](./config.md) 
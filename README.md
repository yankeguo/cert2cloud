# cert2cloud

A tool to replicate Kubernetes TLS secrets to cloud providers (阿里云/腾讯云 CDN SSL 证书管理工具)

## Overview

cert2cloud is a Go-based tool that automates the process of uploading and managing SSL/TLS certificates to major Chinese cloud providers, specifically:

- **阿里云 (Aliyun)** - Uploads certificates to Certificate Management Service (CAS) and binds them to CDN domains
- **腾讯云 (QCloud)** - Uploads certificates to SSL Certificate Service and manages certificate deployment across multiple services

The tool reads certificate files (PEM format), validates them, and automatically handles the upload and deployment process to the configured cloud providers.

## Features

- **Multi-cloud Support**: Simultaneously manage certificates across Aliyun and QCloud
- **Certificate Validation**: Validates certificate format, expiration dates, and serial numbers
- **Deduplication**: Checks for existing certificates to avoid duplicates
- **Automatic Deployment**: Binds certificates to CDN domains and other services
- **Certificate Rotation**: Automatically replaces expiring certificates with new ones
- **Configuration-driven**: JSON-based configuration for easy automation
- **Error Handling**: Comprehensive error handling with detailed logging

## Installation

### From Source

```bash
git clone https://github.com/yankeguo/cert2cloud.git
cd cert2cloud
go build -o cert2cloud .
```

### Using Go Install

```bash
go install github.com/yankeguo/cert2cloud@latest
```

## Configuration

Create a `config.json` file with your certificate and cloud provider settings:

```json
{
  "cert": {
    "name_prefix": "my-cert",
    "cert_pem_file": "/path/to/cert.pem",
    "key_pem_file": "/path/to/key.pem"
  },
  "aliyun": {
    "access_key_id_file": "/path/to/aliyun-access-key-id",
    "access_key_secret_file": "/path/to/aliyun-access-key-secret",
    "region_id_file": "/path/to/aliyun-region",
    "cdn_domains": ["example.com", "www.example.com"]
  },
  "qcloud": {
    "secret_id_file": "/path/to/qcloud-secret-id",
    "secret_key_file": "/path/to/qcloud-secret-key",
    "resource_types": ["cdn","cos"],
    "resource_regions": {
      "cos": ["ap-beijing", "ap-shanghai"]
    }
  }
}
```

### Configuration Options

#### Certificate Options (`cert`)

- `name_prefix`: Prefix for certificate names in cloud providers
- `cert_pem`: Certificate content in PEM format (alternative to file)
- `cert_pem_file`: Path to certificate file in PEM format
- `key_pem`: Private key content in PEM format (alternative to file)
- `key_pem_file`: Path to private key file in PEM format

#### Aliyun Options (`aliyun`)

- `access_key_id`: Aliyun Access Key ID (alternative to file)
- `access_key_id_file`: Path to file containing Access Key ID
- `access_key_secret`: Aliyun Access Key Secret (alternative to file)
- `access_key_secret_file`: Path to file containing Access Key Secret
- `region_id`: Aliyun region ID (alternative to file)
- `region_id_file`: Path to file containing region ID
- `cdn_domains`: List of CDN domain names to bind the certificate to

#### QCloud Options (`qcloud`)

- `secret_id`: QCloud Secret ID (alternative to file)
- `secret_id_file`: Path to file containing Secret ID
- `secret_key`: QCloud Secret Key (alternative to file)
- `secret_key_file`: Path to file containing Secret Key
- `resource_types`: List of resource types to deploy certificate to (e.g., ["cdn"])
- `resource_regions`: Map of resource types to regions for deployment

## Usage

### Basic Usage

```bash
# Use default config.json
cert2cloud

# Use custom config file
cert2cloud -conf /path/to/config.json
```

### Docker Usage

```bash
# Build Docker image
docker build -t cert2cloud .

# Run with config file
docker run -v /path/to/config:/config cert2cloud -conf /config/config.json
```

## How It Works

1. **Certificate Loading**: Reads and validates the certificate files
2. **Certificate Parsing**: Parses the certificate to extract metadata (serial number, domains, expiration)
3. **Cloud Provider Check**:
   - For Aliyun: Checks if certificate already exists by serial number
   - For QCloud: Checks if certificate exists by domain names and expiration time
4. **Certificate Upload**: Uploads new certificate if not found
5. **Service Binding**:
   - Aliyun: Binds certificate to specified CDN domains
   - QCloud: Deploys certificate to specified resource types and regions
6. **Certificate Cleanup**: QCloud automatically replaces expiring certificates

## Security Considerations

- Store sensitive credentials (API keys) in separate files with appropriate permissions
- Use file-based configuration for credentials rather than inline values
- Ensure certificate files have proper read permissions
- Consider using environment variables or secret management systems for production

## Dependencies

- Go 1.24.2+
- Aliyun SDK for Go
- Tencent Cloud SDK for Go
- yankeguo/rg utility library

## Development

### Project Structure

```
cert2cloud/
├── main.go              # Main entry point
├── main_aliyun.go       # Aliyun-specific implementation
├── main_qcloud.go       # QCloud-specific implementation
├── options.go           # Core configuration structures
├── options_aliyun.go    # Aliyun configuration
├── options_qcloud.go    # QCloud configuration
├── utils.go             # Utility functions
├── config.json          # Example configuration
├── go.mod               # Go module definition
├── Dockerfile           # Docker build configuration
└── cog.toml            # Release management configuration
```

### Building

```bash
# Build for current platform
go build -o cert2cloud .

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o cert2cloud-linux-amd64 .
GOOS=darwin GOARCH=amd64 go build -o cert2cloud-darwin-amd64 .
GOOS=windows GOARCH=amd64 go build -o cert2cloud-windows-amd64.exe .
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed history of changes and releases.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/yankeguo/cert2cloud).

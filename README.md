# go-pownmyhash

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

A powerful cross-platform password cracking automation tool that leverages hashcat with smart dictionary management and adaptive attack strategies.

![Go-PownMyHash Banner](https://github.com/1mm0rt41PC/go-pownMyHash/raw/refs/heads/main/img/logo.png)

## Features

- **Automatic Hash Detection**: Identifies common hash types from your input file
- **Smart Dictionary Ranking**: Prioritizes dictionaries based on previous success rates
- **Comprehensive Attack Strategies**: 
  - Historical password analysis
  - Custom dictionaries generation
  - Rule-based attacks
  - Dictionary stacking
  - Brute force attacks
- **Adaptive Learning**: Continuously improves as you crack more passwords
- **Cross-Platform**: Works on both Windows and Linux
- **Auto-Installation**: Downloads and configures hashcat and dependencies automatically
- **Pre-configured**: Comes with optimized rules and popular wordlists

## Prerequisites

- Windows or Linux operating system
- NVIDIA GPU (recommended) or CPU
- 7zip (for dictionary and rule extraction)
- CUDA drivers (for NVIDIA GPU acceleration)

## Installation

1. Download the latest release from the [releases page](https://github.com/yourusername/go-pownmyhash/releases)

2. For Linux, make the binary executable:
   ```bash
   chmod +x go-pownmyhash
   ```

3. Run the tool for the first time to automatically download and configure hashcat:
   ```bash
   ./go-pownmyhash -hashes your_hashes.txt -type auto
   ```

## Usage

### Basic Usage

```bash
./go-pownmyhash -hashes your_hashes.txt
```

### Command Line Options

```
  -dicts string
        Path to dictionaries directory (default "dico")
  -dict-stats string
        Path to dictionary statistics file (default "dict-stats.json")
  -fake
        Fake flag to bypass hashcat installation
  -hashes string
        Path to the hash file
  -historical string
        Path to historical dictionary (default "pownMyHash.dico")
  -rules string
        Path to rules directory
  -type string
        Hash type (auto, ntlm, net-ntlm, net-ntlmv2, krb5tgs$23, dcc2, or numeric mode) (default "auto")
```

### Supported Hash Types

The tool can automatically detect:
- NTLM (SAM format)
- NetNTLMv2
- NetNTLM
- Kerberos 5 TGS-REP
- MS Cache v2
- MD5
- SHA1
- SHA2-256
- SHA2-512
- MySQL SHA1

### Interactive Menu

When you run the tool, you'll be presented with an interactive menu:

```
Choose the order of attack:
1. Historical dictionary
2. Custom dictionary
3. Dictionaries (ranked by efficiency)
4. Brute force password with len=8 with automask
5. Brute force password with len=8
6. Rules stacking with best64 rule on Historical Dict
7. Rules stacking with best64 rule on all dico
```

You can enter the order in which to run these attacks, e.g., `1,2,3,4`.

## How It Works

1. **Dictionary Ranking**: 
   - Each successful password crack is tracked per dictionary
   - Dictionaries with higher success rates are tried first
   - Statistics are saved in a JSON file for future use

2. **Historical Learning**:
   - All cracked passwords are stored in a historical dictionary
   - This dictionary is used as a knowledge base for future attacks
   - The tool applies rules to this historical data to find patterns

3. **Adaptive Strategy**:
   - After each successful dictionary or rule, custom dictionaries are generated
   - The tool recursively applies rules on new passwords for maximum effectiveness

## Dictionary Files

- Dictionaries should be placed in the `dico` directory
- The tool accepts both `.txt` and `.dico` files
- Compressed `.7z` dictionaries are automatically extracted

## Rule Files

- Rules should be placed in the `rules` directory
- The tool will download and configure popular rule sets automatically

## Building From Source

```bash
git clone https://github.com/yourusername/go-pownmyhash.git
cd go-pownmyhash
go build -o go-pownmyhash
```

## Legal Notice

This tool is intended for security professionals to test systems they own or have permission to test. Do not use this for illegal purposes.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

### GPL v3 License Summary

The GPL v3 license guarantees your freedom to:
- Use the software for any purpose
- Change the software to suit your needs
- Share the software with your friends and neighbors
- Share the changes you make

For more details about the GNU GPL v3, visit the [GNU website](https://www.gnu.org/licenses/gpl-3.0.en.html).
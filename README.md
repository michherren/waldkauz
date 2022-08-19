<p align="center">
  <img alt="Waldkauz Logo" src="icon/icon.png" height="140" />
  <h3 align="center">Waldkauz</h3>
</p>

---

This is a weekend-project to explore multi-arch builds and packaging in go. Waldkauz is a small wrapper around the awesome [Redpanda Console (formerly known as Kowl)](https://github.com/redpanda-data/console) Kafka-Client. The generated static binary includes everything needed to run console.

**important:** This is provided as is and only very limited testing was done!

## Installation
Head over to the releases page https://github.com/michherren/waldkauz/releases and download the latest build.

### For Mac Homebrew Users
```
brew tap michherren/waldkauz
brew install waldkauz
```
The waldkauz installation directory with the configuration-file can be found here:
/opt/homebrew/Cellar/waldkauz/X.Y.Z/bin/waldkauz-data

## Acknowledgement
Redpanda for the awesome kafka-client console: https://github.com/redpanda-data/console

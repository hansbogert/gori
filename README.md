# Gori

Gori was created because there was a need for getting the status of git directories inside a projects folder.


## Install

```sh
go install github.com/hansbogert/gori/cmd/gori@latest
```

Or clone and install locally:

```sh
git clone github.com:hansbogert/gori.git && cd gori
go install ./cmd/gori
```

## Usage

```
gori

Emoji Legend:
  🚧: Dirty working directory
  🗄️: Stashed changes
  📤: Not upstreamed

foo1: 🚧🗄️
that-other-project: 🚧
etc: 🚧
microservice1: 🚧
microservice2: 🚧🗄️
microservice3: 🚧
k8s: 🚧
k9s: 📤
rook: 🚧🗄️
sample-controller: 🚧
vagrant-libvirt: 🚧
```
## Missing features

Gori is highly opinionated

- Assumes flat projects dir, so not a multi-level tree
- many more small things :D

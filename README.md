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

## Challenges

Gori relies on `go-git`. Further, Gori relies on git status of the `go-git`
client. At moment of writing the `status` functionality of go-git is (very) slow
compared to cgit. The reason for this is that on every status invocation go-git
rehashes everything.

To circumvent the slow implementation a fork has been made of go-git in which a
fastpath has been implemented in the case that if a workdir file has the same
modification time as it's corresponding file in the Git index, then we re-use
the hash from the index. This is roughly what CGit does.

It seems multiple projects are hampered by the slow status implementation of go-git:

- https://github.com/go-git/go-git/pull/1694 (mine)
- https://github.com/go-git/go-git/pull/1747


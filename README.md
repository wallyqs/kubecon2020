
[NATS](https://nats.io) is a simple, secure and performant communications system for digital systems, services and devices. NATS is part of the Cloud Native Computing Foundation ([CNCF](https://cncf.io)). NATS has over [30 client language implementations](https://nats.io/download/), and its server can run on-premise, in the cloud, at the edge, and even on a Raspberry Pi. NATS can secure and simplify design and operation of modern distributed systems.

## OSCON Demo

This docker image is in support of a talk at [OSCON 2019](https://conferences.oreilly.com/oscon/oscon-or/schedule/2019-07-18).
All the code here is OSS, and the github repo can be found [here](https://github.com/ConnectEverything/oscon2019).

```
> docker run -ti --rm synadia/oscon
```

This will place you in a small alpine linux container that has some NATS utilities (nats-pub, nats-sub, nats-req, and nsc). There is also a simple chat application that is powered by NATS and [NGS](https://synadia.com/ngs/), in a totally secure, and totally distributed way.

## Getting Started

The image comes with some credentials that can be view via `creds-show` helper. The helper uses the `nsc` utility to describe and manipulate users and acounts. These credentials only allow a small set of interactions, which is to request broader permissions to the system.

The nats utilities allow a user to send and receive messages from the global NGS system. These messages are secured and isolated to the [OSCON](https://api.synadia.io/jwt/v1/accounts/AAOSCON6ID63VZPPAZRHMHKNYLNX7N4J5UEWVSI64XLRZXZCYYVBTXG5?decode=true) account. Think of accounts as a secure by default, run anywhere VPC for NATS. By default only other users in your account can receive messages that are sent.

Currently our credentials are too limiting to do much, but feel free to try some `nats-pub` and `nats-sub` calls.

To get our broader permissions we will use `nats-req` to send a secure request and ask for broader credentials to interact with the chat application.
The request to `chat.req.access` will be the username you want to use for the chat application, e.g. derek. We will direct the credentials to a file.

```
> nats-req chat.req.access name > chat.creds
```

You can inspect these permissions with the `nsc` tool.
```
> nsc describe jwt -f chat.creds
```

When running the chat application, our mini Slack clone, you will enter msgs and press enter to send. <TAB\> will move you to select a new channel or to DM others that are online.

```
> chat -creds chat.creds
```

# driftbottle
This is a demo for blockchain study, based on Tendermint. Tendermint bring us a consensus algorithms as BFT-based PoS, visit its [official website](https://tendermint.com/docs/) for details.

This project includes the following simple functions:
1. throw a driftbottle
2. Fishing drifting bottles
3. After that, the launchers and salvagers can communicate with each other.

You can type `dbcli --help` on the command line to view all client commands and related descriptions.

Concepts that developers can learn or need to focus on:
1. Ed25519 Signature Authentication
2. curve25519 key agreement
3. Serialization: protobuf, amino
4. Interactive process of client, Tendermint core and ABCIserver
5. Voting process of BTF-based PoS

Later, I will write a blog to give a more detailed description. And again, this demo is only for learning, practicality and efficiency, etc, are not considered too much.

---
这是一个示例项目，用于区块链学习。该项目基于Tendermint，Tendermint使用PoS实现了拜占庭容错机制，具体可参看其[官方文档](https://tendermint.com/docs/)。

这个项目包括以下简单功能：
1. 扔漂流瓶
2. 捞漂流瓶
3. 之后投放者和打捞者可以相互传递信息

你可以在命令行输入`dbcli --help`查看所有客户端命令和相关描述。

开发人员可以学到或需要关注的概念：
1. ed25519签名认证
2. curve25519密钥协商
3. 序列化：protobuf、amino
4. client、Tendermint core、ABCIserver交互流程
5. BTF-based PoS的投票流程

以后我会另写一篇博文做更细节的描述。再次说明，本示例只是为了学习，实用性和效率等方面并未考虑太多。

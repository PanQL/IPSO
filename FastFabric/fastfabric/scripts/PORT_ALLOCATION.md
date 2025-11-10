# FastFabric 节点端口分配方案

## 端口分配总览

### Orderer 节点
- Orderer Address: `7050`
- Operations: `8443`

### Storage Peer 节点
- Peer Listen: `7051`
- Chaincode: `7052` 
- Operations: `9443`
- 数据目录: `/var/hyperledger/production/storage`

### Endorser Peer 节点  
- Peer Listen: `7053`
- Chaincode: `7054`
- Operations: `10443`
- 数据目录: `/var/hyperledger/production/endorser`
- 配置文件: `core-endorser.yaml`

### FastPeer 节点
- Peer Listen: `7055`
- Chaincode: `7056`
- Operations: `10444`
- 数据目录: `/var/hyperledger/production/fastpeer`

## 启动顺序

1. **Orderer**: `./run_orderer.sh`
2. **Storage Peer**: `./run_storage.sh`  
3. **Endorser Peer**: `./run_endorser.sh`
4. **FastPeer**: `./run_fastpeer.sh`

## 端口检查命令

```bash
# 检查所有 FastFabric 端口
lsof -i :7050 -i :7051 -i :7052 -i :7053 -i :7054 -i :7055 -i :7056 -i :8443 -i :9443 -i :10443 -i :10444

# 检查特定节点端口
lsof -i :7053 -i :7054 -i :10443  # Endorser
lsof -i :7051 -i :7052 -i :9443   # Storage  
lsof -i :7055 -i :7056 -i :10444  # FastPeer
```

## 配置文件说明

- **Storage Peer**: 使用默认 `core.yaml`
- **Endorser Peer**: 使用专用 `core-endorser.yaml`，通过 `FABRIC_CFG_PATH` 指定
- **FastPeer**: 使用默认 `core.yaml`，通过环境变量覆盖端口

## 注意事项

1. 每个节点使用独立的数据目录，避免数据冲突
2. Endorser 使用专用配置文件确保端口不被覆盖
3. Gossip bootstrap 都指向 Storage Peer (7051)
4. 所有节点在同一台机器上运行时端口必须唯一

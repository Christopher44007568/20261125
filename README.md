# 20261125

## 法线锥背面剔除参考实现

这个 Go 模块用可复现的参考场景比较两类背面剔除策略：

- 保守剔除：使用完整顶点法线集生成参考包围锥，只在相机严格位于“保证背面”区域时剔除。
- 激进剔除：使用错误顶点集、单个顶点法线或未经镜像方向修正的轴，并在点积为零时也剔除。

### 参考包围锥

对单位法线集合 `N`，先计算：

```text
axis = normalize(sum(n for n in N))
halfAngle = max(acos(dot(axis, n)) for n in N)
```

该锥包含所有输入法线。若法线为空、非有限值、非单位长度或长度接近零，则参考锥无效；保守策略会拒绝剔除。若法线和为零但每条法线有效，则生成半角 `pi` 的全空间锥，同样不会剔除。

保守剔除的视角向量为：

```text
view = normalize(camera - surfacePoint)
cull iff angle(view, axis) > pi/2 + halfAngle
```

等号表示相机正好落在锥边界上，实现使用严格 `<` 比较，因此边界确定地保留，不依赖浮点误差或迭代顺序。

### 场景

- `planar`：平面 patch，四条法线完全相同，参考半角为 `0`。
- `curved_patch`：60 度对称弯曲 patch，参考半角为 `30` 度；激进策略错误地只取第一个顶点法线。
- `mirrored_patch`：同一弯曲 patch 经过镜像矩阵；参考实现根据负行列式翻转法线方向，激进策略直接变换旧锥轴。
- `degenerate_normal`：包含零法线和相反有效法线；参考锥失效，激进策略仍盲目使用第一个有效法线。

相机使用确定性 Fibonacci 球面采样，并额外加入精确边界视角。误杀率定义为：

```text
falseKillRate = 被该策略剔除且 ground truth 可见的样本数 / ground truth 可见样本数
```

### 运行

```bash
go test ./...
go run ./cmd/normalcone-report -samples 1024
```

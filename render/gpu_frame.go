package render

type drawBatchKey struct {
	texture      *Image
	lightTexture *Image
	options      DrawTrianglesOptions
}

type drawBatch struct {
	key        drawBatchKey
	firstIndex uint32
	indexCount uint32
}

type drawFrame struct {
	floats  []float32
	indices []uint32
	batches []drawBatch
}

type worldFrame struct {
	floats  []float32
	indices []uint32
	batches []drawBatch
}

type worldFrameScratch struct {
	floats     []float32
	indices    []uint32
	batches    []drawBatch
	pending    []pendingWorldBatch
	batchByKey map[drawBatchKey]int
	alpha      []WorldCommand
}

type pendingWorldBatch struct {
	key      drawBatchKey
	commands []WorldCommand
}

func (r *gpuRenderer) buildWorldFrame(screen *Frame) worldFrame {
	commandCount, vertexCount, indexCount := worldFrameCounts(screen.worldCommands)
	scratch := &r.worldFrameScratch
	scratch.floats = reserveSlice(scratch.floats, vertexCount*worldVertexFloatCount)
	scratch.indices = reserveSlice(scratch.indices, indexCount)
	scratch.batches = reserveSlice(scratch.batches, commandCount)
	scratch.pending = reservePendingWorldBatches(scratch.pending, commandCount)
	for index := range scratch.pending {
		scratch.pending[index].commands = scratch.pending[index].commands[:0]
	}
	scratch.pending = scratch.pending[:0]
	scratch.alpha = reserveSlice(scratch.alpha, commandCount)
	scratch.batchByKey = clearDrawBatchKeyMap(scratch.batchByKey, commandCount)
	for _, cmd := range screen.worldCommands {
		if cmd.Texture == nil || cmd.Texture.pix == nil {
			continue
		}
		if !cmd.Options.DepthWrite {
			scratch.alpha = append(scratch.alpha, cmd)
			continue
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		key := drawBatchKey{texture: cmd.Texture, lightTexture: cmd.LightTexture, options: cmd.Options}
		batchIndex, ok := scratch.batchByKey[key]
		if !ok {
			scratch.pending = append(scratch.pending, pendingWorldBatch{key: key})
			batchIndex = len(scratch.pending) - 1
			scratch.batchByKey[key] = batchIndex
		}
		batch := &scratch.pending[batchIndex]
		batch.commands = append(batch.commands, cmd)
	}
	frame := worldFrame{
		floats:  scratch.floats,
		indices: scratch.indices,
		batches: scratch.batches,
	}
	for _, batch := range scratch.pending {
		if len(batch.commands) == 0 {
			continue
		}
		firstIndex := uint32(len(frame.indices))
		indexCountBefore := len(frame.indices)
		for _, cmd := range batch.commands {
			w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
			if w <= 0 || h <= 0 {
				continue
			}
			lw, lh := lightTextureSize(cmd.LightTexture, w, h)
			frame.floats, frame.indices = appendWorldCommand(frame.floats, frame.indices, cmd, w, h, lw, lh)
		}
		indexCount := len(frame.indices) - indexCountBefore
		if indexCount == 0 {
			continue
		}
		frame.batches = append(frame.batches, drawBatch{
			key:        batch.key,
			firstIndex: firstIndex,
			indexCount: uint32(indexCount),
		})
	}
	var current *drawBatch
	for _, cmd := range scratch.alpha {
		key := drawBatchKey{texture: cmd.Texture, lightTexture: cmd.LightTexture, options: cmd.Options}
		if current == nil || current.key != key {
			frame.batches = append(frame.batches, drawBatch{key: key, firstIndex: uint32(len(frame.indices))})
			current = &frame.batches[len(frame.batches)-1]
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		indexCountBefore := len(frame.indices)
		lw, lh := lightTextureSize(cmd.LightTexture, w, h)
		frame.floats, frame.indices = appendWorldCommand(frame.floats, frame.indices, cmd, w, h, lw, lh)
		current.indexCount += uint32(len(frame.indices) - indexCountBefore)
	}
	scratch.floats = frame.floats
	scratch.indices = frame.indices
	scratch.batches = frame.batches
	return frame
}

func reservePendingWorldBatches(values []pendingWorldBatch, capacity int) []pendingWorldBatch {
	if cap(values) < capacity {
		return make([]pendingWorldBatch, 0, capacity)
	}
	return values
}

func reserveSlice[T any](values []T, capacity int) []T {
	if cap(values) < capacity {
		return make([]T, 0, capacity)
	}
	return values[:0]
}

func clearDrawBatchKeyMap(values map[drawBatchKey]int, capacity int) map[drawBatchKey]int {
	if values == nil {
		return make(map[drawBatchKey]int, capacity)
	}
	clear(values)
	return values
}

func appendWorldCommand(floats []float32, indices []uint32, cmd WorldCommand, width, height, lightWidth, lightHeight int) ([]float32, []uint32) {
	base := uint32(len(floats) / worldVertexFloatCount)
	invW, invH := 1/float32(width), 1/float32(height)
	invLW, invLH := 1/float32(lightWidth), 1/float32(lightHeight)
	depthBias := saneDepthBias(cmd.Options.DepthBias)
	flags := worldVertexFlags(cmd.Options)
	for _, v := range cmd.Vertices {
		floats = append(floats,
			v.X, v.Y, v.Z,
			v.SrcX*invW, v.SrcY*invH,
			saneColor(v.ColorR), saneColor(v.ColorG), saneColor(v.ColorB), saneColor(v.ColorA),
			v.DepthX, v.DepthY, v.DepthZ,
			depthBias,
			flags,
			v.LightSrcX*invLW, v.LightSrcY*invLH,
		)
	}
	for _, idx := range cmd.Indices {
		indices = append(indices, base+uint32(idx))
	}
	return floats, indices
}

// worldVertexFlags packs per-vertex switches into one float: bit 0 enables
// fog, bit 1 the environment colour.
func worldVertexFlags(options DrawTrianglesOptions) float32 {
	flags := float32(0)
	if !options.DisableFog {
		flags++
	}
	if options.Environment {
		flags += 2
	}
	return flags
}

func worldMeshGPUData(mesh *WorldMesh, width, height int) ([]float32, []uint32) {
	floats := make([]float32, 0, len(mesh.vertices)*worldVertexFloatCount)
	indices := make([]uint32, 0, len(mesh.indices))
	cmd := WorldCommand{
		Vertices:     mesh.vertices,
		Indices:      mesh.indices,
		Texture:      mesh.texture,
		LightTexture: mesh.lightTexture,
		Options:      mesh.options,
	}
	lw, lh := lightTextureSize(mesh.lightTexture, width, height)
	return appendWorldCommand(floats, indices, cmd, width, height, lw, lh)
}

func lightTextureSize(texture *Image, fallbackWidth, fallbackHeight int) (int, int) {
	if texture == nil || texture.pix == nil {
		return fallbackWidth, fallbackHeight
	}
	w, h := texture.Bounds().Dx(), texture.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return fallbackWidth, fallbackHeight
	}
	return w, h
}

func worldFrameCounts(commands []WorldCommand) (commandsOut, vertices, indices int) {
	for _, cmd := range commands {
		if cmd.Texture == nil || cmd.Texture.pix == nil {
			continue
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		commandsOut++
		vertices += len(cmd.Vertices)
		indices += len(cmd.Indices)
	}
	return commandsOut, vertices, indices
}

func (r *gpuRenderer) buildFrame(screen *Frame) drawFrame {
	commandCount, vertexCount, indexCount := frameCounts(screen.commands)
	frame := drawFrame{
		floats:  make([]float32, 0, vertexCount*screenVertexFloatCount),
		indices: make([]uint32, 0, indexCount),
		batches: make([]drawBatch, 0, commandCount),
	}
	var current *drawBatch
	for _, cmd := range screen.commands {
		if cmd.Texture == nil || cmd.Texture.pix == nil {
			continue
		}
		key := drawBatchKey{texture: cmd.Texture, options: cmd.Options}
		if current == nil || current.key != key {
			frame.batches = append(frame.batches, drawBatch{key: key, firstIndex: uint32(len(frame.indices))})
			current = &frame.batches[len(frame.batches)-1]
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		indexCountBefore := len(frame.indices)
		frame.floats, frame.indices = appendDrawCommand(frame.floats, frame.indices, cmd, w, h, screen.screenScaleX, screen.screenScaleY)
		current.indexCount += uint32(len(frame.indices) - indexCountBefore)
	}
	return frame
}

func appendDrawCommand(floats []float32, indices []uint32, cmd DrawCommand, width, height int, scaleX, scaleY float32) ([]float32, []uint32) {
	if scaleX <= 0 {
		scaleX = 1
	}
	if scaleY <= 0 {
		scaleY = 1
	}
	base := uint32(len(floats) / screenVertexFloatCount)
	invW, invH := 1/float32(width), 1/float32(height)
	for _, v := range cmd.Vertices {
		floats = append(floats,
			v.DstX*scaleX, v.DstY*scaleY,
			v.SrcX*invW, v.SrcY*invH,
			saneColor(v.ColorR), saneColor(v.ColorG), saneColor(v.ColorB), saneColor(v.ColorA),
		)
	}
	for _, idx := range cmd.Indices {
		indices = append(indices, base+uint32(idx))
	}
	return floats, indices
}

func frameCounts(commands []DrawCommand) (commandsOut, vertices, indices int) {
	for _, cmd := range commands {
		if cmd.Texture == nil || cmd.Texture.pix == nil {
			continue
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		commandsOut++
		vertices += len(cmd.Vertices)
		indices += len(cmd.Indices)
	}
	return commandsOut, vertices, indices
}

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

type pendingWorldBatch struct {
	key     drawBatchKey
	floats  []float32
	indices []uint32
}

func (r *gpuRenderer) buildWorldFrame(screen *Image) worldFrame {
	commandCount, vertexCount, indexCount := worldFrameCounts(screen.worldCommands)
	pending := make([]pendingWorldBatch, 0, commandCount)
	batchByKey := make(map[drawBatchKey]int, commandCount)
	alpha := make([]WorldCommand, 0, commandCount)
	for _, cmd := range screen.worldCommands {
		if cmd.Texture == nil || cmd.Texture.pix == nil {
			continue
		}
		if !cmd.Options.DepthWrite {
			alpha = append(alpha, cmd)
			continue
		}
		key := drawBatchKey{texture: cmd.Texture, lightTexture: cmd.LightTexture, options: cmd.Options}
		batchIndex, ok := batchByKey[key]
		if !ok {
			pending = append(pending, pendingWorldBatch{key: key})
			batchIndex = len(pending) - 1
			batchByKey[key] = batchIndex
		}
		w, h := cmd.Texture.Bounds().Dx(), cmd.Texture.Bounds().Dy()
		if w <= 0 || h <= 0 {
			continue
		}
		batch := &pending[batchIndex]
		lw, lh := lightTextureSize(cmd.LightTexture, w, h)
		batch.floats, batch.indices = appendWorldCommand(batch.floats, batch.indices, cmd, w, h, lw, lh)
	}
	frame := worldFrame{
		floats:  make([]float32, 0, vertexCount*worldVertexFloatCount),
		indices: make([]uint32, 0, indexCount),
		batches: make([]drawBatch, 0, commandCount),
	}
	for _, batch := range pending {
		if len(batch.indices) == 0 || len(batch.floats) == 0 {
			continue
		}
		vertexBase := uint32(len(frame.floats) / worldVertexFloatCount)
		firstIndex := uint32(len(frame.indices))
		frame.floats = append(frame.floats, batch.floats...)
		for _, index := range batch.indices {
			frame.indices = append(frame.indices, vertexBase+index)
		}
		frame.batches = append(frame.batches, drawBatch{
			key:        batch.key,
			firstIndex: firstIndex,
			indexCount: uint32(len(batch.indices)),
		})
	}
	var current *drawBatch
	for _, cmd := range alpha {
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
	return frame
}

func appendWorldCommand(floats []float32, indices []uint32, cmd WorldCommand, width, height, lightWidth, lightHeight int) ([]float32, []uint32) {
	base := uint32(len(floats) / worldVertexFloatCount)
	invW, invH := 1/float32(width), 1/float32(height)
	invLW, invLH := 1/float32(lightWidth), 1/float32(lightHeight)
	depthBias := saneDepthBias(cmd.Options.DepthBias)
	fogEnabled := float32(1)
	if cmd.Options.DisableFog {
		fogEnabled = 0
	}
	for _, v := range cmd.Vertices {
		floats = append(floats,
			v.X, v.Y, v.Z,
			v.SrcX*invW, v.SrcY*invH,
			saneColor(v.ColorR), saneColor(v.ColorG), saneColor(v.ColorB), saneColor(v.ColorA),
			v.DepthX, v.DepthY, v.DepthZ,
			depthBias,
			fogEnabled,
			v.LightSrcX*invLW, v.LightSrcY*invLH,
		)
	}
	for _, idx := range cmd.Indices {
		indices = append(indices, base+uint32(idx))
	}
	return floats, indices
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

func (r *gpuRenderer) buildFrame(screen *Image) drawFrame {
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

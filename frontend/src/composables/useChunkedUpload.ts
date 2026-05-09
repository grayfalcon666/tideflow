import { ref, readonly } from 'vue'
import * as videoService from '../services/video'

export interface ChunkedUploadOptions {
  chunkSize?: number   // 分片大小，默认 5MB
  concurrency?: number // 并发数，默认 3
  maxRetries?: number  // 每片最大重试次数，默认 3
}

export type UploadStatus = 'idle' | 'uploading' | 'merging' | 'done' | 'error'

export interface UploadResult {
  playUrl: string
  duration: number
  width: number
  height: number
  isVertical: boolean
}

export function useChunkedUpload(options: ChunkedUploadOptions = {}) {
  const chunkSize = options.chunkSize ?? 5 * 1024 * 1024
  const concurrency = options.concurrency ?? 3
  const maxRetries = options.maxRetries ?? 3

  const progress = ref(0)
  const speed = ref('')
  const status = ref<UploadStatus>('idle')
  const error = ref('')

  let aborted = false
  let speedTimer: ReturnType<typeof setInterval> | null = null

  function cancel() {
    aborted = true
    if (speedTimer) clearInterval(speedTimer)
    status.value = 'idle'
  }

  function formatSpeed(bytesPerSec: number): string {
    if (bytesPerSec < 1024) return `${bytesPerSec} B/s`
    if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
    return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`
  }

  async function uploadWithRetry(
    uploadId: string,
    index: number,
    chunk: Blob,
    retries: number,
  ): Promise<void> {
    for (let attempt = 0; attempt <= retries; attempt++) {
      if (aborted) throw new Error('aborted')
      try {
        await videoService.uploadChunk(uploadId, index, chunk)
        return
      } catch (e) {
        if (attempt === retries) throw e
        const delay = Math.min(1000 * 2 ** attempt, 10000)
        await new Promise((r) => setTimeout(r, delay))
      }
    }
  }

  async function upload(file: File): Promise<UploadResult> {
    aborted = false
    error.value = ''
    status.value = 'uploading'
    progress.value = 0
    speed.value = ''

    const totalChunks = Math.ceil(file.size / chunkSize)
    let uploadedBytes = 0
    const startTime = Date.now()

    speedTimer = setInterval(() => {
      const elapsed = (Date.now() - startTime) / 1000
      if (elapsed > 0) {
        speed.value = formatSpeed(uploadedBytes / elapsed)
      }
    }, 1000)

    try {
      // 1) Init session
      const initResp = await videoService.initChunkedUpload(
        file.name,
        file.size,
        chunkSize,
      )
      const uploadId = initResp.data.data!.upload_id

      // 2) Build pending queue
      const pending: number[] = []
      for (let i = 0; i < totalChunks; i++) {
        pending.push(i)
      }

      // 3) Worker pool
      const workers: Promise<void>[] = []
      for (let w = 0; w < concurrency; w++) {
        workers.push(
          (async () => {
            while (pending.length > 0 && !aborted) {
              const index = pending.shift()!
              const start = index * chunkSize
              const end = Math.min(start + chunkSize, file.size)
              const chunk = file.slice(start, end)

              await uploadWithRetry(uploadId, index, chunk, maxRetries)

              uploadedBytes += end - start
              progress.value = Math.round(
                ((totalChunks - pending.length) / totalChunks) * 100,
              )
            }
          })(),
        )
      }

      await Promise.all(workers)

      if (aborted) {
        throw new Error('aborted')
      }

      // 4) Complete
      status.value = 'merging'
      progress.value = 99
      const completeResp =
        await videoService.completeChunkedUpload(uploadId)
      const data = completeResp.data.data!

      status.value = 'done'
      progress.value = 100

      return {
        playUrl: data.play_url,
        duration: data.duration,
        width: data.width,
        height: data.height,
        isVertical: data.is_vertical,
      }
    } catch (e: any) {
      if (e.message !== 'aborted') {
        error.value = e.response?.data?.message || e.message || '上传失败'
        status.value = 'error'
      }
      throw e
    } finally {
      if (speedTimer) {
        clearInterval(speedTimer)
        speedTimer = null
      }
    }
  }

  async function resume(
    uploadId: string,
    file: File,
  ): Promise<UploadResult> {
    aborted = false
    error.value = ''
    status.value = 'uploading'
    progress.value = 0
    speed.value = ''

    const totalChunks = Math.ceil(file.size / chunkSize)
    let uploadedBytes = 0
    const startTime = Date.now()

    speedTimer = setInterval(() => {
      const elapsed = (Date.now() - startTime) / 1000
      if (elapsed > 0 && uploadedBytes > 0) {
        speed.value = formatSpeed(uploadedBytes / elapsed)
      }
    }, 1000)

    try {
      // 1) Query which chunks are already uploaded
      const statusResp = await videoService.getUploadStatus(uploadId)
      const session = statusResp.data.data!
      const uploadedSet = new Set(session.uploaded_chunks)

      // 2) Build pending queue (skip uploaded)
      const pending: number[] = []
      for (let i = 0; i < totalChunks; i++) {
        if (!uploadedSet.has(i)) {
          pending.push(i)
        }
      }

      // Pre-compute uploaded bytes
      uploadedBytes = uploadedSet.size * chunkSize

      // 3) Worker pool
      const workers: Promise<void>[] = []
      for (let w = 0; w < concurrency; w++) {
        workers.push(
          (async () => {
            while (pending.length > 0 && !aborted) {
              const index = pending.shift()!
              const start = index * chunkSize
              const end = Math.min(start + chunkSize, file.size)
              const chunk = file.slice(start, end)

              await uploadWithRetry(uploadId, index, chunk, maxRetries)

              uploadedBytes += end - start
              // progress = (total - pending) / total
              const uploadedCount =
                uploadedSet.size + (totalChunks - pending.length)
              progress.value = Math.round(
                (uploadedCount / totalChunks) * 100,
              )
            }
          })(),
        )
      }

      await Promise.all(workers)

      if (aborted) throw new Error('aborted')

      // 4) Complete
      status.value = 'merging'
      progress.value = 99
      const completeResp =
        await videoService.completeChunkedUpload(uploadId)
      const data = completeResp.data.data!

      status.value = 'done'
      progress.value = 100

      return {
        playUrl: data.play_url,
        duration: data.duration,
        width: data.width,
        height: data.height,
        isVertical: data.is_vertical,
      }
    } catch (e: any) {
      if (e.message !== 'aborted') {
        error.value = e.response?.data?.message || e.message || '上传失败'
        status.value = 'error'
      }
      throw e
    } finally {
      if (speedTimer) {
        clearInterval(speedTimer)
        speedTimer = null
      }
    }
  }

  return {
    progress: readonly(progress),
    speed: readonly(speed),
    status: readonly(status),
    error: readonly(error),
    upload,
    resume,
    cancel,
  }
}

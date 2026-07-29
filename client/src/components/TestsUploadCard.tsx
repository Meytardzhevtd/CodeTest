import { useRef, useState } from 'react'
import type { DragEvent } from 'react'
import { ApiError, tasksApi } from '../api'

type Status = 'idle' | 'uploading' | 'success' | 'error'

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`
  return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

export function TestsUploadCard({ taskId }: { taskId: string }) {
  const [file, setFile] = useState<File | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [status, setStatus] = useState<Status>('idle')
  const [message, setMessage] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const pickFile = (candidate: File | undefined) => {
    if (!candidate) return
    if (!candidate.name.toLowerCase().endsWith('.zip')) {
      setStatus('error')
      setMessage('Нужен zip-архив (.zip)')
      return
    }
    setFile(candidate)
    setStatus('idle')
    setMessage('')
  }

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    setDragActive(false)
    pickFile(event.dataTransfer.files[0])
  }

  const handleUpload = async () => {
    if (!file) return
    setStatus('uploading')
    setMessage('')

    try {
      const res = await tasksApi.uploadTests(taskId, file)
      setStatus('success')
      setMessage(`Загружено тестов: ${res.tests_uploaded}`)
      setFile(null)
    } catch (err) {
      setStatus('error')
      setMessage(err instanceof ApiError ? err.message : 'Не удалось загрузить тесты')
    }
  }

  return (
    <section className="card tests-upload">
      <div className="tests-upload-header">
        <h2>Тесты</h2>
        <p className="muted">
          Zip-архив с файлами вида <code>test_001.in</code> / <code>test_001.out</code>. Новый
          архив полностью заменит текущие тесты задачи.
        </p>
      </div>

      <div
        className={`dropzone${dragActive ? ' dropzone-active' : ''}${file ? ' dropzone-has-file' : ''}`}
        onClick={() => inputRef.current?.click()}
        onDragOver={(event) => {
          event.preventDefault()
          setDragActive(true)
        }}
        onDragLeave={() => setDragActive(false)}
        onDrop={handleDrop}
        role="button"
        tabIndex={0}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') inputRef.current?.click()
        }}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".zip"
          hidden
          onChange={(event) => pickFile(event.target.files?.[0])}
        />

        {file ? (
          <div className="dropzone-file">
            <span className="dropzone-file-icon" aria-hidden="true">
              📦
            </span>
            <div>
              <div className="dropzone-file-name">{file.name}</div>
              <div className="dropzone-file-size">{formatSize(file.size)}</div>
            </div>
            <button
              type="button"
              className="dropzone-clear"
              aria-label="Убрать файл"
              onClick={(event) => {
                event.stopPropagation()
                setFile(null)
                setStatus('idle')
                setMessage('')
                if (inputRef.current) inputRef.current.value = ''
              }}
            >
              ✕
            </button>
          </div>
        ) : (
          <>
            <span className="dropzone-icon" aria-hidden="true">
              ⬆
            </span>
            <p>
              Перетащите zip-архив сюда или <span className="link-btn">выберите файл</span>
            </p>
          </>
        )}
      </div>

      {status === 'error' ? <div className="alert alert-error">{message}</div> : null}
      {status === 'success' ? <div className="alert alert-success">{message}</div> : null}

      <div className="tests-upload-footer">
        <button
          type="button"
          className="btn btn-primary"
          disabled={!file || status === 'uploading'}
          onClick={handleUpload}
        >
          {status === 'uploading' ? 'Загрузка...' : 'Загрузить тесты'}
        </button>
      </div>
    </section>
  )
}

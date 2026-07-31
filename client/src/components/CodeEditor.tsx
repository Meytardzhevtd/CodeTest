import { useEffect, useMemo, useState } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import type { Extension } from '@codemirror/state'
import { python } from '@codemirror/lang-python'
import { cpp } from '@codemirror/lang-cpp'
import { StreamLanguage } from '@codemirror/language'
import { go } from '@codemirror/legacy-modes/mode/go'
import { githubLight, githubDark } from '@uiw/codemirror-theme-github'

const LANGUAGE_EXTENSIONS: Record<string, Extension> = {
  python: python(),
  go: StreamLanguage.define(go),
  cpp: cpp(),
}

function usePrefersDark(): boolean {
  const [prefersDark, setPrefersDark] = useState(
    () => window.matchMedia('(prefers-color-scheme: dark)').matches,
  )

  useEffect(() => {
    const query = window.matchMedia('(prefers-color-scheme: dark)')
    const listener = (event: MediaQueryListEvent) => setPrefersDark(event.matches)
    query.addEventListener('change', listener)
    return () => query.removeEventListener('change', listener)
  }, [])

  return prefersDark
}

export function CodeEditor({
  value,
  onChange,
  language,
}: {
  value: string
  onChange: (value: string) => void
  language: string
}) {
  const prefersDark = usePrefersDark()
  const extensions = useMemo(
    () => [LANGUAGE_EXTENSIONS[language] ?? python()],
    [language],
  )

  return (
    <CodeMirror
      className="code-editor"
      value={value}
      onChange={onChange}
      extensions={extensions}
      theme={prefersDark ? githubDark : githubLight}
      basicSetup={{ tabSize: 4 }}
      minHeight="320px"
    />
  )
}

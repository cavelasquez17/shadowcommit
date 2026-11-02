import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { renderMarkdown } from '@/lib/format'
import { applyCopium } from '@/lib/copium'
import { useApp } from '@/store/app'

/**
 * Markdown renders sanitised post bodies. Input is HTML-escaped before any markup is
 * generated (see renderMarkdown), so dangerouslySetInnerHTML is safe here.
 *
 * The click handler intercepts internal @mention links so they route through the SPA
 * instead of triggering a full page load.
 */
export function Markdown({ source, className = '' }: { source: string; className?: string }) {
  const navigate = useNavigate()
  const { copium } = useApp()
  const html = useMemo(() => renderMarkdown(copium ? applyCopium(source) : source), [source, copium])

  return (
    <div
      className={`prose-sc ${className}`}
      onClick={(e) => {
        const anchor = (e.target as HTMLElement).closest('a')
        const href = anchor?.getAttribute('href')
        if (href?.startsWith('/')) {
          e.preventDefault()
          void navigate(href)
        }
      }}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

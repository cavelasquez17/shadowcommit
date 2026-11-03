import { Component, type ErrorInfo, type ReactNode } from 'react'
import { useLocation, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

interface Props {
  children: ReactNode
}
interface State {
  error: Error | null
}

/**
 * Catches a render error in its subtree and shows a message instead of unmounting the whole app.
 * Without this, one page's bug blanks the navigation and the footer too.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Keep the failure visible in the console; swallowing it would hide the bug.
    console.error(error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return <CrashedPage onRetry={() => this.setState({ error: null })} />
    }
    return this.props.children
  }
}

function CrashedPage({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation()
  return (
    <div role="alert" className="slab mx-auto mt-10 flex max-w-xl flex-col items-center gap-4 p-10 text-center">
      <span className="text-4xl grayscale">💀</span>
      <h2 className="text-xl font-black tracking-tight">{t('errorBoundary.title')}</h2>
      <p className="max-w-md text-xs text-zinc-500">{t('errorBoundary.body')}</p>
      <div className="flex flex-wrap justify-center gap-3">
        <button type="button" className="btn" onClick={onRetry}>
          {t('common.retry')}
        </button>
        <Link to="/" className="btn btn-ghost">
          {t('errorBoundary.home')}
        </Link>
      </div>
    </div>
  )
}

/**
 * Resets when the user navigates: keying by path remounts the boundary, so a broken page does not
 * stay broken on the next one.
 */
export function PageBoundary({ children }: { children: ReactNode }) {
  const location = useLocation()
  return <ErrorBoundary key={location.pathname}>{children}</ErrorBoundary>
}

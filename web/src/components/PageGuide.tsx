import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { FeatureBanner } from '@/components/FeatureBanner'

const MAX_TIPS = 5

/**
 * Lays a page out beside its guide: the banner is a side column on desktop and stacks above the
 * page on mobile. The copy comes from guides.<page>; a page with no guide renders unchanged.
 */
export function PageGuide({ page, children }: { page: string; children: ReactNode }) {
  const { t, i18n } = useTranslation()
  const tips: string[] = []
  for (let n = 1; n <= MAX_TIPS; n++) {
    const key = `guides.${page}.tip${n}`
    if (i18n.exists(key)) tips.push(t(key))
  }
  if (!i18n.exists(`guides.${page}.title`) || tips.length === 0) return <>{children}</>

  return (
    <div className="lg:flex lg:items-start lg:gap-6">
      <aside className="mb-6 lg:sticky lg:top-4 lg:mb-0 lg:w-64 lg:shrink-0">
        <FeatureBanner id={page} title={t(`guides.${page}.title`)} tips={tips} />
      </aside>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}

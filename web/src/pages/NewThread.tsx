import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Eye, PenLine, Send } from 'lucide-react'
import { api, ApiError, type Category } from '@/lib/api'
import { useApp } from '@/store/app'
import { SectionTitle } from '@/components/ui'
import { Select } from '@/pages/Feed'
import { MentionTextarea } from '@/components/MentionTextarea'
import { Markdown } from '@/components/Markdown'

const MIN_TITLE = 8
const MIN_BODY = 20

export function NewThread() {
  const { t, i18n } = useTranslation()
  const { user, meta } = useApp()
  const navigate = useNavigate()
  const [params] = useSearchParams()

  const [categories, setCategories] = useState<Category[]>([])
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [category, setCategory] = useState(params.get('category') ?? '')
  const [lang, setLang] = useState(i18n.language)
  const [tags, setTags] = useState('')
  const [sarcasm, setSarcasm] = useState(100)
  const [nsfw, setNsfw] = useState(false)
  const [preview, setPreview] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void api.categories().then((r) => {
      setCategories(r.items)
      setCategory((c) => c || r.items[0]?.slug || '')
    })
  }, [])

  // Anyone who lands here without a session gets bounced home rather than shown a dead form.
  useEffect(() => {
    if (user === null) void navigate('/')
  }, [user, navigate])

  const valid = title.trim().length >= MIN_TITLE && body.trim().length >= MIN_BODY && category

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError(null)
    try {
      const { thread } = await api.createThread({
        title: title.trim(),
        body: body.trim(),
        category,
        language: lang,
        sarcasm,
        nsfw_corporate: nsfw,
        tags: tags
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
          .slice(0, 5),
      })
      void navigate(`/t/${thread.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const selected = categories.find((c) => c.slug === category)

  return (
    <div className="mx-auto max-w-3xl">
      <SectionTitle color="#ccff00" sub={t('compose.subtitle')}>
        {t('compose.title')}
      </SectionTitle>

      <form onSubmit={submit} className="space-y-4">
        <div className="slab p-4" style={{ borderColor: selected?.color ?? '#ccff00' }}>
          <label className="block">
            <span className="mb-1 block text-[10px] tracking-widest text-zinc-500 uppercase">
              {t('compose.fieldTitle')}
            </span>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t('compose.titlePlaceholder')}
              maxLength={200}
              className="field text-base font-bold"
            />
          </label>
          <p className="mt-1 text-right text-[10px] text-zinc-600 tabular-nums">
            {title.length}/200
            {title.trim().length < MIN_TITLE && (
              <span className="ml-2 text-[var(--color-blood)]">
                {t('compose.charsLeft', { count: MIN_TITLE })}
              </span>
            )}
          </p>
        </div>

        <div className="slab p-4">
          <div className="mb-2 flex items-center justify-between gap-2">
            <span className="text-[10px] tracking-widest text-zinc-500 uppercase">
              {t('compose.fieldBody')}
            </span>
            <div className="flex gap-1">
              <button
                type="button"
                onClick={() => setPreview(false)}
                className="btn px-2 py-1 text-[10px]"
                style={
                  !preview
                    ? { background: '#ccff00', borderColor: '#000', color: '#000' }
                    : { background: 'var(--color-slab)', borderColor: 'var(--color-edge)', color: '#a1a1aa' }
                }
              >
                <PenLine size={11} strokeWidth={3} /> {t('compose.write')}
              </button>
              <button
                type="button"
                onClick={() => setPreview(true)}
                className="btn px-2 py-1 text-[10px]"
                style={
                  preview
                    ? { background: '#00e5ff', borderColor: '#000', color: '#000' }
                    : { background: 'var(--color-slab)', borderColor: 'var(--color-edge)', color: '#a1a1aa' }
                }
              >
                <Eye size={11} strokeWidth={3} /> {t('compose.preview')}
              </button>
            </div>
          </div>

          {preview ? (
            <div className="min-h-52 border-2 border-[var(--color-edge)] bg-[var(--color-void)] p-3">
              {body.trim() ? (
                <Markdown source={body} />
              ) : (
                <p className="text-xs text-zinc-600">{t('compose.bodyPlaceholder')}</p>
              )}
            </div>
          ) : (
            <MentionTextarea
              value={body}
              onChange={setBody}
              rows={14}
              placeholder={t('compose.bodyPlaceholder')}
            />
          )}

          <p className="mt-1 text-right text-[10px] text-zinc-600 tabular-nums">
            {body.length}
            {body.trim().length < MIN_BODY && (
              <span className="ml-2 text-[var(--color-blood)]">
                {t('compose.charsLeft', { count: MIN_BODY })}
              </span>
            )}
          </p>
        </div>

        <div className="slab grid gap-4 p-4 sm:grid-cols-2">
          <Select
            label={t('compose.fieldCategory')}
            value={category}
            onChange={setCategory}
            options={categories.map((c) => ({ value: c.slug, label: `${c.emoji} ${c.name}` }))}
          />
          <Select
            label={t('compose.fieldLanguage')}
            value={lang}
            onChange={setLang}
            options={(meta?.languages ?? []).map((l) => ({
              value: l.value,
              label: `${l.emoji} ${l.label}`,
            }))}
          />

          <label className="block sm:col-span-2">
            <span className="mb-1 block text-[10px] tracking-widest text-zinc-500 uppercase">
              {t('compose.fieldTags')}
            </span>
            <input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder={t('compose.tagsPlaceholder')}
              className="field"
            />
          </label>

          <label className="block sm:col-span-2">
            <span className="mb-1 flex items-center justify-between text-[10px] tracking-widest text-zinc-500 uppercase">
              {t('compose.fieldSarcasm')}
              <span className="font-black text-[var(--color-bubble)] tabular-nums">{sarcasm}%</span>
            </span>
            <input
              type="range"
              min={0}
              max={100}
              value={sarcasm}
              onChange={(e) => setSarcasm(Number(e.target.value))}
              className="w-full accent-[var(--color-bubble)]"
            />
            <span className="mt-1 block text-[10px] text-zinc-600">{t('compose.sarcasmHint')}</span>
          </label>

          <label className="flex cursor-pointer items-center gap-2 sm:col-span-2">
            <input
              type="checkbox"
              checked={nsfw}
              onChange={(e) => setNsfw(e.target.checked)}
              className="h-4 w-4 accent-[var(--color-blood)]"
            />
            <span className="text-xs text-zinc-400">⚠ {t('compose.nsfwLabel')}</span>
          </label>
        </div>

        {error && (
          <p className="slab border-[var(--color-blood)] p-3 text-xs text-[var(--color-blood)]">
            {error}
          </p>
        )}

        <div className="flex justify-end">
          <button type="submit" disabled={!valid || busy} className="btn btn-primary px-6 py-3">
            <Send size={14} strokeWidth={2.5} />
            {busy ? t('compose.submitting') : t('compose.submit')}
          </button>
        </div>
      </form>
    </div>
  )
}

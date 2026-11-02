import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, Calculator, RefreshCw, Skull } from 'lucide-react'
import { SectionTitle, StatBox } from '@/components/ui'

/**
 * RiskCalculator is a viral traffic tool. Users answer 8 questions about their
 * role, tooling, and company. The algorithm computes a "Replacement Risk Score"
 * 0-100 with a shareable verdict. No backend needed — pure client-side math.
 *
 * The scoring is deliberately tongue-in-cheek but grounded in real signals:
 * - How much of your work is pattern-matching vs novel problem-solving
 * - Whether your company has an "AI initiative" (the writing on the wall)
 * - How much of your output is reviewed by a human
 * - Seniority (juniors are more exposed; principals less so)
 */
export function RiskCalculator() {
  const { t } = useTranslation()
  const [answers, setAnswers] = useState<Record<string, number>>({})
  const [submitted, setSubmitted] = useState(false)

  const questions = useMemo(() => [
    {
      id: 'ai_usage',
      label: t('risk.q1'),
      options: [
        { label: t('risk.q1a1'), value: 0 },
        { label: t('risk.q1a2'), value: 15 },
        { label: t('risk.q1a3'), value: 30 },
        { label: t('risk.q1a4'), value: 45 },
      ],
    },
    {
      id: 'task_type',
      label: t('risk.q2'),
      options: [
        { label: t('risk.q2a1'), value: 40 },
        { label: t('risk.q2a2'), value: 25 },
        { label: t('risk.q2a3'), value: 10 },
        { label: t('risk.q2a4'), value: 0 },
      ],
    },
    {
      id: 'ai_initiative',
      label: t('risk.q3'),
      options: [
        { label: t('risk.q3a1'), value: 0 },
        { label: t('risk.q3a2'), value: 10 },
        { label: t('risk.q3a3'), value: 20 },
        { label: t('risk.q3a4'), value: 35 },
      ],
    },
    {
      id: 'human_review',
      label: t('risk.q4'),
      options: [
        { label: t('risk.q4a1'), value: 0 },
        { label: t('risk.q4a2'), value: 10 },
        { label: t('risk.q4a3'), value: 20 },
        { label: t('risk.q4a4'), value: 30 },
      ],
    },
    {
      id: 'seniority',
      label: t('risk.q5'),
      options: [
        { label: t('risk.q5a1'), value: 35 },
        { label: t('risk.q5a2'), value: 25 },
        { label: t('risk.q5a3'), value: 15 },
        { label: t('risk.q5a4'), value: 5 },
      ],
    },
    {
      id: 'domain',
      label: t('risk.q6'),
      options: [
        { label: t('risk.q6a1'), value: 30 },
        { label: t('risk.q6a2'), value: 20 },
        { label: t('risk.q6a3'), value: 10 },
        { label: t('risk.q6a4'), value: 0 },
      ],
    },
    {
      id: 'meetings',
      label: t('risk.q7'),
      options: [
        { label: t('risk.q7a1'), value: 30 },
        { label: t('risk.q7a2'), value: 15 },
        { label: t('risk.q7a3'), value: 5 },
        { label: t('risk.q7a4'), value: 0 },
      ],
    },
    {
      id: 'company_size',
      label: t('risk.q8'),
      options: [
        { label: t('risk.q8a1'), value: 20 },
        { label: t('risk.q8a2'), value: 15 },
        { label: t('risk.q8a3'), value: 10 },
        { label: t('risk.q8a4'), value: 5 },
      ],
    },
  ], [t])

  const score = useMemo(() => {
    const total = Object.values(answers).reduce((a, b) => a + b, 0)
    return Math.min(100, total)
  }, [answers])

  const allAnswered = Object.keys(answers).length === questions.length

  const verdict = useMemo(() => {
    if (score < 20) return { emoji: '🛡️', color: '#39ff88', text: t('risk.verdictSafe') }
    if (score < 40) return { emoji: '🤔', color: '#00e5ff', text: t('risk.verdictLow') }
    if (score < 60) return { emoji: '⚠️', color: '#ffe14d', text: t('risk.verdictMedium') }
    if (score < 80) return { emoji: '🫠', color: '#ff8a1f', text: t('risk.verdictHigh') }
    return { emoji: '💀', color: '#ff3355', text: t('risk.verdictCritical') }
  }, [score, t])

  function select(qid: string, value: number) {
    setAnswers((a) => ({ ...a, [qid]: value }))
  }

  function reset() {
    setAnswers({})
    setSubmitted(false)
  }

  function share() {
    const text = `My Replacement Risk Score: ${score}/100 ${verdict.emoji}\n\n${verdict.text}\n\nCalculate yours at ShadowCommit`
    if (navigator.share) {
      void navigator.share({ text, url: window.location.href })
    } else {
      void navigator.clipboard.writeText(`${text} — ${window.location.href}`)
    }
  }

  return (
    <div className="space-y-8">
      <SectionTitle color="#ff3355" sub={t('risk.subtitle')}>
        {t('risk.title')}
      </SectionTitle>

      {!submitted ? (
        <div className="space-y-6">
          <p className="max-w-2xl text-sm leading-relaxed text-zinc-400">
            {t('risk.intro')}
          </p>

          {questions.map((q, qi) => (
            <div key={q.id} className="slab p-4">
              <h3 className="text-xs font-black tracking-wider uppercase text-zinc-300">
                <span className="text-[var(--color-blood)]">{qi + 1}.</span> {q.label}
              </h3>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                {q.options.map((opt) => (
                  <button
                    key={opt.label}
                    onClick={() => select(q.id, opt.value)}
                    className={`border-2 px-3 py-2 text-left text-[12px] transition-all ${
                      answers[q.id] === opt.value
                        ? 'border-[var(--color-toxic)] bg-[color-mix(in_srgb,var(--color-toxic)_10%,transparent)] text-white'
                        : 'border-[var(--color-edge)] text-zinc-400 hover:border-zinc-600'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          ))}

          <button
            className="btn btn-primary w-full"
            disabled={!allAnswered}
            onClick={() => setSubmitted(true)}
          >
            <Calculator size={14} strokeWidth={3} /> {t('risk.calculate')}
          </button>
          {!allAnswered && (
            <p className="text-center text-[10px] text-zinc-600">
              {t('risk.answerAll')}
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-6">
          {/* Score display */}
          <div
            className="slab p-8 text-center"
            style={{ borderColor: verdict.color, boxShadow: `6px 6px 0 0 color-mix(in srgb, ${verdict.color} 30%, transparent)` }}
          >
            <div className="text-6xl">{verdict.emoji}</div>
            <div className="mt-4 text-7xl font-black tracking-tighter" style={{ color: verdict.color }}>
              {score}
              <span className="text-2xl text-zinc-600">/100</span>
            </div>
            <p className="mt-3 text-sm font-bold tracking-wide" style={{ color: verdict.color }}>
              {verdict.text}
            </p>
          </div>

          {/* Score breakdown */}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <StatBox value={score < 40 ? 'LOW' : score < 70 ? 'MID' : 'HIGH'} label={t('risk.riskLevel')} color={verdict.color} />
            <StatBox value={`${Object.keys(answers).length}/8`} label={t('risk.questionsAnswered')} color="#00e5ff" />
            <StatBox value={score < 50 ? t('risk.humanSafe') : t('risk.machineBait')} label={t('risk.outlook')} color={score < 50 ? '#39ff88' : '#ff3355'} />
            <StatBox value="∞" label={t('risk.copium')} color="#a855f7" />
          </div>

          {/* Verdict paragraph */}
          <div className="slab p-4" style={{ borderColor: 'var(--color-edge)' }}>
            <div className="flex items-start gap-3">
              <AlertTriangle size={16} className="mt-0.5 shrink-0 text-[var(--color-butter)]" />
              <p className="text-[13px] leading-relaxed text-zinc-300">
                {score < 20 && t('risk.analysisSafe')}
                {score >= 20 && score < 40 && t('risk.analysisLow')}
                {score >= 40 && score < 60 && t('risk.analysisMedium')}
                {score >= 60 && score < 80 && t('risk.analysisHigh')}
                {score >= 80 && t('risk.analysisCritical')}
              </p>
            </div>
          </div>

          {/* Actions */}
          <div className="flex flex-wrap gap-3">
            <button className="btn btn-primary" onClick={share}>
              {t('risk.share')}
            </button>
            <button className="btn btn-ghost" onClick={reset}>
              <RefreshCw size={12} strokeWidth={3} /> {t('risk.retake')}
            </button>
            <a href="/feed" className="btn btn-ghost">
              <Skull size={12} strokeWidth={2.5} /> {t('risk.readStories')}
            </a>
          </div>
        </div>
      )}
    </div>
  )
}

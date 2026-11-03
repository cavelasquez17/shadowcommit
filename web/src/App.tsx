import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { Landing } from '@/pages/Landing'
import { Feed } from '@/pages/Feed'
import { ThreadDetail } from '@/pages/ThreadDetail'
import { NewThread } from '@/pages/NewThread'
import { Profile } from '@/pages/Profile'
import { Members } from '@/pages/Members'
import { Wall } from '@/pages/Wall'
import { Onboarding } from '@/pages/Onboarding'
import { Invites, Join } from '@/pages/Invites'
import { Jobs } from '@/pages/Jobs'
import { RiskCalculator } from '@/pages/RiskCalculator'
import { Notifications } from '@/pages/Notifications'
import { EmptyState } from '@/components/ui'
import { PageGuide } from '@/components/PageGuide'

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Landing />} />
        <Route path="feed" element={<PageGuide page="feed"><Feed /></PageGuide>} />
        <Route path="t/:id" element={<ThreadDetail />} />
        <Route path="new" element={<PageGuide page="compose"><NewThread /></PageGuide>} />
        <Route path="u/:username" element={<Profile />} />
        <Route path="members" element={<PageGuide page="members"><Members /></PageGuide>} />
        <Route path="wall" element={<PageGuide page="wall"><Wall /></PageGuide>} />
        <Route path="onboarding" element={<Onboarding />} />
        <Route path="invites" element={<PageGuide page="invites"><Invites /></PageGuide>} />
        <Route path="join" element={<Join />} />
        <Route path="jobs" element={<PageGuide page="jobs"><Jobs /></PageGuide>} />
        <Route path="risk" element={<PageGuide page="risk"><RiskCalculator /></PageGuide>} />
        <Route path="notifications" element={<Notifications />} />
        {/* Legacy/alias paths people will guess. */}
        <Route path="threads" element={<Navigate to="/feed" replace />} />
        <Route
          path="*"
          element={
            <EmptyState emoji="🧭">
              404. This page was deprecated by an intern with an LLM.
            </EmptyState>
          }
        />
      </Route>
    </Routes>
  )
}

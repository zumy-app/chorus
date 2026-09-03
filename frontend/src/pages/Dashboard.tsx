import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'
import { useStore } from '../store'
import { chatAPI, learningAPI, api } from '../services/api'
import type { Chat, LearningDashboard, CallSession } from '@chorus/shared'

export default function Dashboard() {
  const navigate = useNavigate()
  const user = useStore(s => s.user)
  const [chats, setChats] = useState<Chat[]>([])
  const [calls, setCalls] = useState<CallSession[]>([])
  const [learning, setLearning] = useState<LearningDashboard | null>(null)
  const [loading, setLoading] = useState(true)

  const targetLanguage = user?.targetLanguages?.[0] ?? 'es'
  const nativeLanguage = user?.nativeLanguage ?? 'en'

  useEffect(() => {
    let active = true
    setLoading(true)
    Promise.allSettled([
      chatAPI.getChats(),
      api.get<CallSession[]>('/calls/history?limit=6').then(r => r.data).catch(() => [] as CallSession[]),
      learningAPI.getDashboard(targetLanguage, nativeLanguage).catch(() => null),
    ]).then(results => {
      if (!active) return
      const c = results[0].status === 'fulfilled' ? (results[0].value as Chat[]) : []
      const h = results[1].status === 'fulfilled' ? (results[1].value as unknown as CallSession[]) : []
      const d = results[2].status === 'fulfilled' ? (results[2].value as LearningDashboard | null) : null
      setChats(Array.isArray(c) ? c : [])
      setCalls(Array.isArray(h) ? h : [])
      if (d) setLearning(d)
      setLoading(false)
    })
    return () => { active = false }
  }, [targetLanguage, nativeLanguage])

  const unreadTotal = chats.reduce((s, c) => s + (c.unreadCount ?? 0), 0)
  const pct = learning?.dailyGoal.percent ?? 0

  return (
    <div data-testid="dashboard-page" className="min-h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto pb-24 md:pb-6">
        <div className="max-w-[1280px] mx-auto w-full px-4 md:px-6 lg:px-8 py-6">
          <header className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
            <div>
              <h1 className="font-headline-lg text-headline-lg text-on-background">Dashboard</h1>
              <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">Calls, chats, and learning — in one control center.</p>
            </div>
            <div className="flex gap-2">
              <button onClick={() => navigate('/chat')} className="bg-primary text-on-primary font-label-md text-label-md px-5 py-2.5 rounded-full hover:bg-primary/90 transition">Open Chats</button>
              <button onClick={() => navigate('/learn')} className="bg-surface-container-high text-on-surface font-label-md text-label-md px-5 py-2.5 rounded-full hover:bg-surface-container transition">Learn</button>
            </div>
          </header>

          {loading ? (
            <p className="font-body-md text-body-md text-on-surface-variant">Loading dashboard...</p>
          ) : (
            <>
              <section className="grid grid-cols-2 lg:grid-cols-4 gap-3 md:gap-4 mb-6">
                <div className="bg-surface-container-lowest rounded-2xl p-4 md:p-5 shadow-sm border border-outline-variant/20 flex flex-col gap-2">
                  <div className="w-9 h-9 rounded-full bg-primary-container/20 flex items-center justify-center text-primary"><span className="material-symbols-outlined text-[20px]">forum</span></div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{chats.length}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">Active chats</div>
                  {unreadTotal > 0 && <div className="font-label-sm text-label-sm text-primary">{unreadTotal} unread</div>}
                </div>
                <div className="bg-surface-container-lowest rounded-2xl p-4 md:p-5 shadow-sm border border-outline-variant/20 flex flex-col gap-2">
                  <div className="w-9 h-9 rounded-full bg-secondary-container/20 flex items-center justify-center text-secondary"><span className="material-symbols-outlined text-[20px]">call</span></div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{calls.length}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">Recent calls</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant">{calls.filter(c=>c.status==='active').length} active</div>
                </div>
                <div className="bg-surface-container-lowest rounded-2xl p-4 md:p-5 shadow-sm border border-outline-variant/20 flex flex-col gap-2">
                  <div className="w-9 h-9 rounded-full bg-tertiary-container/15 flex items-center justify-center text-tertiary"><span className="material-symbols-outlined text-[20px]">local_fire_department</span></div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{learning?.streak.days ?? 0}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">Day streak</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant">{learning?.dailyGoal.completedItems ?? 0} / {learning?.dailyGoal.targetItems ?? 0} today</div>
                </div>
                <div className="bg-surface-container-lowest rounded-2xl p-4 md:p-5 shadow-sm border border-outline-variant/20 flex flex-col gap-2">
                  <div className="w-9 h-9 rounded-full bg-primary-container/20 flex items-center justify-center text-primary"><span className="material-symbols-outlined text-[20px]">menu_book</span></div>
                  <div className="font-headline-lg text-headline-lg text-on-surface">{learning?.vocabulary.total ?? 0}</div>
                  <div className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider">Vocabulary</div>
                  <div className="font-label-sm text-label-sm text-outline">{learning?.vocabulary.dueToday ?? 0} due today</div>
                </div>
              </section>

              <section className="grid grid-cols-1 lg:grid-cols-12 gap-4 md:gap-6">
                <div data-testid="dashboard-chats-panel" className="lg:col-span-4 bg-surface-container-lowest rounded-2xl shadow-sm border border-outline-variant/20 p-5 flex flex-col">
                  <div className="flex items-center justify-between mb-4">
                    <h2 className="font-headline-sm text-headline-sm text-on-surface flex items-center gap-2"><span className="material-symbols-outlined text-primary text-[20px]">chat</span> Chats</h2>
                    <button onClick={() => navigate('/chat')} className="font-label-sm text-label-sm text-primary hover:underline">View all</button>
                  </div>
                  {chats.length === 0 ? (
                    <p className="font-body-sm text-body-sm text-on-surface-variant">No chats yet. Start a conversation to see it here.</p>
                  ) : (
                    <ul className="flex flex-col divide-y divide-outline-variant/20">
                      {chats.slice(0, 6).map(chat => {
                        const name = chat.type === 'group' ? (chat.name || 'Group') : (chat.participants.find(p=>p.userId!==user?.id)?.user?.displayName || chat.participants.find(p=>p.userId!==user?.id)?.user?.username || 'Direct chat')
                        return (
                          <li key={chat.id} className="py-3 flex items-center gap-3 cursor-pointer hover:bg-surface-container-low rounded-xl px-2 -mx-2 transition" onClick={() => navigate('/chat')}>
                            <div className="w-9 h-9 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center font-bold text-sm shrink-0">{name.charAt(0).toUpperCase()}</div>
                            <div className="flex-1 min-w-0">
                              <div className="font-label-md text-label-md text-on-surface truncate">{name}</div>
                              <div className="font-body-sm text-body-sm text-on-surface-variant truncate">{chat.lastMessage?.text || 'No messages yet'}</div>
                            </div>
                            {(chat.unreadCount ?? 0) > 0 && <span className="bg-primary text-on-primary text-xs font-bold px-2 py-0.5 rounded-full">{chat.unreadCount}</span>}
                          </li>
                        )
                      })}
                    </ul>
                  )}
                  <div className="mt-4 pt-4 border-t border-outline-variant/20">
                    <div className="w-full h-2 bg-surface-container-high rounded-full overflow-hidden"><div className="h-full bg-primary rounded-full transition-all" style={{ width: `${Math.min(100, chats.length * 18)}%` }} /></div>
                    <p className="font-label-sm text-label-sm text-on-surface-variant mt-2">{chats.length} conversations</p>
                  </div>
                </div>

                <div data-testid="dashboard-calls-panel" className="lg:col-span-4 bg-surface-container-lowest rounded-2xl shadow-sm border border-outline-variant/20 p-5 flex flex-col">
                  <div className="flex items-center justify-between mb-4">
                    <h2 className="font-headline-sm text-headline-sm text-on-surface flex items-center gap-2"><span className="material-symbols-outlined text-secondary text-[20px]">video_call</span> Calls</h2>
                    <button onClick={() => navigate('/chat')} className="font-label-sm text-label-sm text-primary hover:underline">Start call</button>
                  </div>
                  {calls.length === 0 ? (
                    <div className="flex-1 flex flex-col items-center justify-center py-8 gap-3">
                      <div className="w-12 h-12 rounded-full bg-surface-container flex items-center justify-center text-outline"><span className="material-symbols-outlined">call</span></div>
                      <p className="font-body-sm text-body-sm text-on-surface-variant text-center">No recent calls. Start an audio or video call from any chat.</p>
                      <button onClick={() => navigate('/chat')} className="mt-2 bg-secondary-container text-on-secondary-container font-label-md text-label-md px-4 py-2 rounded-full">Go to chats</button>
                    </div>
                  ) : (
                    <ul className="flex flex-col gap-2">
                      {calls.slice(0, 6).map(call => (
                        <li key={call.id} className="flex items-center justify-between p-3 rounded-xl bg-surface-container/60 border border-transparent hover:border-outline-variant/30 transition">
                          <div className="flex items-center gap-3 min-w-0">
                            <div className={`w-8 h-8 rounded-full flex items-center justify-center ${call.type === 'video' ? 'bg-secondary-container text-on-secondary-container' : 'bg-primary-container text-on-primary-container'}`}>
                              <span className="material-symbols-outlined text-[18px]">{call.type === 'video' ? 'videocam' : 'call'}</span>
                            </div>
                            <div className="min-w-0">
                              <div className="font-label-md text-label-md text-on-surface capitalize">{call.type} call</div>
                              <div className="font-label-sm text-label-sm text-on-surface-variant truncate">{new Date(call.startedAt).toLocaleString()} · {call.status}</div>
                            </div>
                          </div>
                          <span className={`text-xs px-2 py-1 rounded-full font-medium ${call.status === 'active' ? 'bg-tertiary-container text-on-tertiary-container' : 'bg-surface-container-high text-on-surface-variant'}`}>{call.status}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                  <div className="mt-auto pt-4 flex gap-2">
                    <button onClick={() => navigate('/chat')} className="flex-1 bg-primary text-on-primary font-label-md text-label-md py-2.5 rounded-full">New call</button>
                    <button onClick={() => navigate('/search')} className="flex-1 bg-surface-container-high text-on-surface font-label-md text-label-md py-2.5 rounded-full">Search calls</button>
                  </div>
                </div>

                <div data-testid="dashboard-learning-panel" className="lg:col-span-4 flex flex-col gap-4">
                  <div className="bg-surface-container-lowest rounded-2xl shadow-sm border border-outline-variant/20 p-5">
                    <div className="flex items-center justify-between mb-3">
                      <h2 className="font-headline-sm text-headline-sm text-on-surface flex items-center gap-2"><span className="material-symbols-outlined text-tertiary text-[20px]">school</span> Learning</h2>
                      <button onClick={() => navigate('/learn')} className="font-label-sm text-label-sm text-primary hover:underline">Open Learn</button>
                    </div>
                    {learning ? (
                      <>
                        <div className="flex items-center gap-4">
                          <div className="relative w-20 h-20 shrink-0">
                            <svg className="w-full h-full -rotate-90" viewBox="0 0 100 100">
                              <circle cx="50" cy="50" r="42" fill="transparent" stroke="currentColor" strokeWidth="10" className="text-surface-container-high" />
                              <circle cx="50" cy="50" r="42" fill="transparent" stroke="currentColor" strokeWidth="10" strokeLinecap="round" strokeDasharray={2 * Math.PI * 42} strokeDashoffset={2 * Math.PI * 42 * (1 - pct / 100)} className="text-primary" />
                            </svg>
                            <div className="absolute inset-0 flex items-center justify-center font-headline-sm text-headline-sm text-primary">{pct}%</div>
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="font-label-md text-label-md text-on-surface">Daily goal</div>
                            <div className="font-body-sm text-body-sm text-on-surface-variant">{learning.dailyGoal.completedItems} / {learning.dailyGoal.targetItems} items</div>
                            <div className="mt-2 flex items-center gap-2 text-secondary"><span className="material-symbols-outlined text-[16px]">local_fire_department</span><span className="font-label-sm text-label-sm">{learning.streak.days} day streak</span></div>
                          </div>
                        </div>
                        <div className="grid grid-cols-2 gap-3 mt-4">
                          <div className="bg-surface-container rounded-xl p-3">
                            <div className="font-label-sm text-label-sm text-on-surface-variant uppercase">Fluency</div>
                            <div className="font-headline-md text-headline-md text-on-surface">{learning.fluency.readinessScore}</div>
                            <div className="font-label-sm text-label-sm text-primary">{learning.fluency.label}</div>
                          </div>
                          <div className="bg-surface-container rounded-xl p-3">
                            <div className="font-label-sm text-label-sm text-on-surface-variant uppercase">Mastered</div>
                            <div className="font-headline-md text-headline-md text-on-surface">{learning.vocabulary.mastered}</div>
                            <div className="font-label-sm text-label-sm text-on-surface-variant">{learning.vocabulary.dueToday} due</div>
                          </div>
                        </div>
                        {learning.recommendedActivities.length > 0 && (
                          <div className="mt-4">
                            <div className="font-label-md text-label-md text-on-surface mb-2">Up next</div>
                            <div className="bg-primary/5 rounded-xl p-3 border border-primary/10">
                              <div className="font-label-md text-label-md text-on-surface">{learning.recommendedActivities[0].title}</div>
                              <div className="font-body-sm text-body-sm text-on-surface-variant line-clamp-2">{learning.recommendedActivities[0].description}</div>
                              <button onClick={() => navigate('/learn')} className="mt-2 bg-primary text-on-primary font-label-sm text-label-sm px-3 py-1.5 rounded-full">Start</button>
                            </div>
                          </div>
                        )}
                      </>
                    ) : (
                      <div className="py-6 text-center">
                        <p className="font-body-sm text-body-sm text-on-surface-variant">Learning data unavailable.</p>
                        <button onClick={() => navigate('/learn')} className="mt-3 bg-primary text-on-primary font-label-md text-label-md px-4 py-2 rounded-full">Go to Learn</button>
                      </div>
                    )}
                  </div>
                  <div className="bg-gradient-to-br from-primary to-secondary rounded-2xl p-5 text-white">
                    <h3 className="font-headline-sm text-headline-sm">Control Center</h3>
                    <p className="font-body-sm text-body-sm opacity-90 mt-1">Wide-screen optimized. Manage chats, calls, and learning without switching tabs.</p>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <button onClick={() => navigate('/chat')} className="bg-white text-primary font-label-md text-label-md px-4 py-2 rounded-full">Chats</button>
                      <button onClick={() => navigate('/learn/roadmap')} className="bg-white/20 text-white font-label-md text-label-md px-4 py-2 rounded-full border border-white/30">Roadmap</button>
                      <button onClick={() => navigate('/learn/vocabulary')} className="bg-white/20 text-white font-label-md text-label-md px-4 py-2 rounded-full border border-white/30">Vocabulary</button>
                    </div>
                  </div>
                </div>
              </section>
            </>
          )}
        </div>
      </main>
      <BottomNav />
    </div>
  )
}

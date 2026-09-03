import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { createApiClient, resolveApiConfig } from '@chorus/shared'
import { Capacitor } from '@capacitor/core'

const platform: 'web' | 'ios' | 'android' = Capacitor.isNativePlatform() ? (Capacitor.getPlatform() as 'ios' | 'android') : 'web'
const { baseURL } = resolveApiConfig({ platform, dev: import.meta.env.DEV, origin: import.meta.env.VITE_API_URL, version: import.meta.env.VITE_API_VERSION })
const storage = { getItem: async (k:string)=>localStorage.getItem(k), setItem: async(k:string,v:string)=>{localStorage.setItem(k,v)}, removeItem: async(k:string)=>{localStorage.removeItem(k)} }
const client = createApiClient({ baseURL, storage })

function highlight(text: string, query: string) {
  if (!query) return text
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx === -1) return text
  return (
    <>
      {text.slice(0, idx)}
      <span className="bg-secondary-fixed/50 px-1 rounded font-medium">{text.slice(idx, idx + query.length)}</span>
      {text.slice(idx + query.length)}
    </>
  )
}

type Filter = 'all' | 'messages' | 'media' | 'people'

export default function UniversalSearch() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const initialQ = params.get('q') || ''
  const [query, setQuery] = useState(initialQ)
  const [filter, setFilter] = useState<Filter>('all')
  const [messages, setMessages] = useState<any[]>([])
  const [media, setMedia] = useState<any[]>([])
  const [chats, setChats] = useState<any[]>([])
  const [contacts, setContacts] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [recent, setRecent] = useState<string[]>(()=>{ try{ return JSON.parse(localStorage.getItem('chorus_recent_searches')||'[]')}catch{return []}})
  const [hasSearched, setHasSearched] = useState(false)

  const saveRecent = useCallback((q:string)=>{
    if(!q.trim())return
    setRecent(prev=>{
      const next=[q, ...prev.filter(x=>x!==q)].slice(0,8)
      localStorage.setItem('chorus_recent_searches', JSON.stringify(next))
      return next
    })
  },[])

  const doSearch = useCallback(async (q:string)=>{
    const trimmed=q.trim()
    if(!trimmed){ setMessages([]); setMedia([]); setChats([]); setContacts([]); return }
    setLoading(true)
    setHasSearched(true)
    saveRecent(trimmed)
    setParams({ q: trimmed }, { replace:true })
    try{
      const results = await Promise.allSettled([
        (filter==='all'||filter==='messages') ? client.search.universal(trimmed) : Promise.resolve({messages:[], media:[], total:0, mediaTotal:0, hasMore:false} as any),
        (filter==='all'||filter==='media') ? client.search.media(trimmed) : Promise.resolve({media:[], total:0, hasMore:false} as any),
        (filter==='all'||filter==='people') ? client.search.chats(trimmed) : Promise.resolve([] as any),
        (filter==='all'||filter==='people') ? client.search.contacts(trimmed) : Promise.resolve([] as any),
      ])
      if(results[0].status==='fulfilled'){ const r:any=results[0].value; setMessages(r.messages||[]); if(filter==='all') setMedia(r.media||[]) }
      if(results[1].status==='fulfilled' && filter!=='all'){ const r:any=results[1].value; setMedia(r.media||[]) }
      if(results[2].status==='fulfilled') setChats(results[2].value as any)
      if(results[3].status==='fulfilled') setContacts(results[3].value as any)
    }catch{}
    setLoading(false)
  },[filter, saveRecent, setParams])

  useEffect(()=>{ if(initialQ) doSearch(initialQ) },[]) // eslint-disable-line

  useEffect(()=>{
    const id=setTimeout(()=>{ if(query.trim()) doSearch(query) }, 320)
    return ()=>clearTimeout(id)
  },[query, filter]) // eslint-disable-line

  const clearRecent = ()=>{ localStorage.removeItem('chorus_recent_searches'); setRecent([]) }

  return (
    <div className="min-h-screen bg-background flex flex-col">
      <header className="sticky top-0 z-30 bg-surface border-b border-outline-variant px-4 py-3 flex items-center gap-3">
        <button onClick={()=>navigate(-1)} className="p-2 hover:bg-surface-variant/20 rounded-full"><span className="material-symbols-outlined">arrow_back</span></button>
        <div className="flex-1 relative">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-outline text-[20px]">search</span>
          <input value={query} onChange={e=>setQuery(e.target.value)} onKeyDown={e=>e.key==='Enter' && doSearch(query)} placeholder={t('search.universalPlaceholder','Search messages, media, or people...')} className="w-full bg-surface-container-low rounded-full py-3 pl-11 pr-10 text-on-surface placeholder:text-outline focus:ring-2 focus:ring-primary focus:outline-none" autoFocus />
          {query && <button onClick={()=>{setQuery(''); setMessages([]); setMedia([]); setChats([]); setContacts([]); setHasSearched(false)}} className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 hover:bg-surface-container rounded-full"><span className="material-symbols-outlined text-[18px] text-outline">close</span></button>}
        </div>
      </header>

      <div className="flex gap-2 px-4 py-3 overflow-x-auto">
        {(['all','messages','media','people'] as Filter[]).map(f=>(
          <button key={f} onClick={()=>setFilter(f)} className={`px-4 py-1.5 rounded-full font-label-md text-label-md flex items-center gap-1.5 whitespace-nowrap ${filter===f ? 'bg-primary text-on-primary' : 'bg-surface-container text-on-surface-variant'}`}>
            <span className="material-symbols-outlined text-[18px]">{f==='messages'?'chat':f==='media'?'image':f==='people'?'person':'search'}</span>{t(`search.filter_${f}`, f)}
          </button>
        ))}
      </div>

      <div className="flex-1 max-w-2xl w-full mx-auto px-4 pb-8 flex flex-col gap-6">
        {!hasSearched && recent.length>0 && (
          <section>
            <div className="flex justify-between items-center mb-2">
              <h2 className="font-label-md text-label-md text-outline uppercase tracking-wide">{t('search.recent','Recent Searches')}</h2>
              <button onClick={clearRecent} className="text-xs text-primary">{t('search.clear','Clear')}</button>
            </div>
            <div className="flex flex-wrap gap-2">
              {recent.map(r=>(
                <button key={r} onClick={()=>{setQuery(r); doSearch(r)}} className="px-3 py-1.5 bg-surface-container-low border border-outline-variant/30 rounded-lg text-on-surface text-sm flex items-center gap-1.5">
                  <span className="material-symbols-outlined text-[16px] text-outline">history</span>{r}
                </button>
              ))}
            </div>
          </section>
        )}

        {loading && <div className="text-center py-8 text-outline">{t('common.searching')}</div>}

        {!loading && hasSearched && messages.length===0 && media.length===0 && chats.length===0 && contacts.length===0 && (
          <div className="text-center py-12 text-on-surface-variant">
            <p className="text-4xl mb-3">📭</p>
            <p>{t('search.noResults','No results for "{{query}}"', {query})}</p>
          </div>
        )}

        {(filter==='all'||filter==='messages') && messages.length>0 && (
          <section>
            <h2 className="font-headline-sm text-headline-sm mb-3">{t('search.resultsFor','Results for "{{query}}"', {query})} — {messages.length} {t('search.messages','messages')}</h2>
            <div className="flex flex-col gap-3">
              {messages.map((m:any)=>(
                <div key={m.id} onClick={()=>navigate(`/chat`)} className="bg-surface-container-lowest p-4 rounded-xl shadow flex gap-4 cursor-pointer hover:shadow-md">
                  <div className="w-10 h-10 rounded-full bg-tertiary-container flex items-center justify-center shrink-0 font-semibold">{(m.sender?.displayName||'?').charAt(0)}</div>
                  <div className="flex-1 min-w-0">
                    <div className="flex justify-between items-baseline mb-1"><span className="font-semibold text-on-surface text-sm truncate">{m.sender?.displayName||m.senderId?.slice(0,8)}</span><span className="text-xs text-outline">{m.timestamp? new Date(m.timestamp).toLocaleDateString(): ''}</span></div>
                    <p className="text-sm text-on-surface-variant line-clamp-2">{highlight(m.text||'', query)}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {(filter==='all'||filter==='media') && media.length>0 && (
          <section>
            <h2 className="font-headline-sm text-headline-sm mb-3">{t('search.mediaTitle','Media')} — {media.length}</h2>
            <div className="flex flex-col gap-3">
              {media.map((a:any)=>(
                <div key={a.id} className="bg-surface-container-lowest p-4 rounded-xl shadow flex gap-4 border-l-4 border-secondary">
                  <div className="w-12 h-12 rounded-lg bg-surface-container-high flex items-center justify-center shrink-0"><span className="material-symbols-outlined">{a.type==='image'?'image':a.type==='video'?'videocam':a.type==='audio'?'graphic_eq':'description'}</span></div>
                  <div className="flex-1 min-w-0">
                    <div className="font-semibold text-sm truncate">{a.fileName||a.type}</div>
                    <div className="text-xs text-primary mb-1">{a.type}</div>
                    <p className="text-sm text-on-surface-variant truncate">{highlight(a.fileName||'', query)}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {(filter==='all'||filter==='people') && (chats.length>0 || contacts.length>0) && (
          <section>
            <h2 className="font-headline-sm text-headline-sm mb-3">{t('search.people','People & Chats')}</h2>
            <div className="flex flex-col gap-2">
              {chats.map((c:any)=>(
                <div key={c.id} onClick={()=>navigate(`/chat`)} className="p-3 rounded-xl bg-surface-container-low flex items-center gap-3 cursor-pointer">
                  <span className="material-symbols-outlined text-outline">chat</span><span className="font-medium text-sm">{c.name||c.id.slice(0,8)}</span>
                </div>
              ))}
              {contacts.map((u:any)=>(
                <div key={u.id} className="p-3 rounded-xl bg-surface-container-low flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-primary-container flex items-center justify-center text-xs font-bold">{u.displayName?.charAt(0)||'?'}</div>
                  <div className="flex-1 min-w-0"><div className="font-medium text-sm truncate">{u.displayName}</div><div className="text-xs text-outline truncate">@{u.username}</div></div>
                </div>
              ))}
            </div>
          </section>
        )}
      </div>
    </div>
  )
}

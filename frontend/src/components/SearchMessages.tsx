import { useState, useRef, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import type { Message, MediaAttachment } from '@chorus/shared'
import { createApiClient, resolveApiConfig } from '@chorus/shared'
import { Capacitor } from '@capacitor/core'
import { formatDistanceToNow } from 'date-fns'

const plat: 'web'|'ios'|'android' = Capacitor.isNativePlatform() ? (Capacitor.getPlatform() as any) : 'web'
const { baseURL } = resolveApiConfig({ platform: plat, dev: import.meta.env.DEV, origin: import.meta.env.VITE_API_URL, version: import.meta.env.VITE_API_VERSION })
const sto = { getItem: async(k:string)=>localStorage.getItem(k), setItem: async(k:string,v:string)=>{localStorage.setItem(k,v)}, removeItem: async(k:string)=>{localStorage.removeItem(k)} }
const client = createApiClient({ baseURL, storage: sto })

type Filter='all'|'messages'|'media'|'people'

export default function SearchMessages({ chatId, onClose, onSelectMessage }: { chatId?: string; onClose: ()=>void; onSelectMessage?:(m:Message)=>void }) {
  const { t } = useTranslation()
  const [query,setQuery]=useState('')
  const [filter,setFilter]=useState<Filter>('all')
  const [msgs,setMsgs]=useState<Message[]>([])
  const [media,setMedia]=useState<MediaAttachment[]>([])
  const [chats,setChats]=useState<any[]>([])
  const [contacts,setContacts]=useState<any[]>([])
  const [loading,setLoading]=useState(false)
  const [hasSearched,setHasSearched]=useState(false)
  const [recent,setRecent]=useState<string[]>(()=>{try{return JSON.parse(localStorage.getItem('chorus_recent_searches')||'[]')}catch{return []}})
  const inputRef=useRef<HTMLInputElement>(null)
  useEffect(()=>{inputRef.current?.focus()},[])
  const saveRecent=useCallback((q:string)=>{ if(!q.trim())return; setRecent(p=>{const n=[q,...p.filter(x=>x!==q)].slice(0,8); localStorage.setItem('chorus_recent_searches', JSON.stringify(n)); return n})},[])
  const doSearch=useCallback(async()=>{
    const q=query.trim()
    if(!q) return
    setLoading(true); setHasSearched(true); saveRecent(q)
    try{
      const [r1,r2,r3,r4]=await Promise.allSettled([
        (filter==='all'||filter==='messages') ? client.search.universal(q, {chatId}) : Promise.resolve({messages:[],media:[],total:0,mediaTotal:0,hasMore:false} as any),
        (filter==='all'||filter==='media') ? client.search.media(q, {chatId}) : Promise.resolve({media:[],total:0,hasMore:false} as any),
        (filter==='all'||filter==='people') ? client.search.chats(q) : Promise.resolve([] as any),
        (filter==='all'||filter==='people') ? client.search.contacts(q) : Promise.resolve([] as any),
      ])
      if(r1.status==='fulfilled'){ const r:any=r1.value; setMsgs(r.messages||[]); if(filter==='all') setMedia(r.media||[]) }
      if(r2.status==='fulfilled' && filter!=='all'){ setMedia((r2.value as any).media||[]) }
      if(r3.status==='fulfilled') setChats(r3.value as any)
      if(r4.status==='fulfilled') setContacts(r4.value as any)
    }catch(e){ console.error(e)} finally{ setLoading(false)}
  },[query,filter,chatId,saveRecent])

  const hl=(text:string)=>{ if(!query) return text; const i=text.toLowerCase().indexOf(query.toLowerCase()); if(i===-1) return text; return <>{text.slice(0,i)}<mark className="bg-secondary-fixed/50 px-0.5 rounded">{text.slice(i,i+query.length)}</mark>{text.slice(i+query.length)}</> }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-start justify-center pt-10 z-50 p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col">
        <div className="p-4 border-b flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <div className="flex-1 relative">
              <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 text-[20px]">search</span>
              <input ref={inputRef} value={query} onChange={e=>setQuery(e.target.value)} onKeyDown={e=>e.key==='Enter'&&doSearch()} placeholder={t('search.universalPlaceholder','Search messages, media, or people...')} className="w-full pl-10 pr-4 py-2.5 border rounded-full focus:ring-2 focus:ring-primary focus:outline-none text-sm" />
            </div>
            <button onClick={doSearch} disabled={loading||!query.trim()} className="px-4 py-2 bg-primary text-on-primary rounded-full text-sm font-medium disabled:opacity-40">{loading? t('common.searching'): t('common.search')}</button>
            <button onClick={onClose} className="text-gray-500 text-xl px-2">×</button>
          </div>
          <div className="flex gap-2 overflow-x-auto">
            {(['all','messages','media','people'] as Filter[]).map(f=>(
              <button key={f} onClick={()=>setFilter(f)} className={`px-3 py-1 rounded-full text-xs font-semibold flex items-center gap-1 whitespace-nowrap ${filter===f?'bg-primary text-white':'bg-gray-100 text-gray-600'}`}>
                <span className="material-symbols-outlined text-[16px]">{f==='messages'?'chat':f==='media'?'image':f==='people'?'person':'search'}</span>{t(`search.filter_${f}`,f)}
              </button>
            ))}
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
          {!hasSearched && recent.length>0 && (
            <div>
              <div className="text-xs uppercase tracking-wide text-gray-400 mb-2">{t('search.recent','Recent Searches')}</div>
              <div className="flex flex-wrap gap-2">{recent.map(r=>(<button key={r} onClick={()=>{setQuery(r); setTimeout(()=>doSearch(),0)}} className="px-2.5 py-1 bg-gray-50 border rounded-lg text-sm flex items-center gap-1"><span className="material-symbols-outlined text-[14px]">history</span>{r}</button>))}</div>
            </div>
          )}
          {!hasSearched && recent.length===0 && <div className="text-center text-gray-400 py-10"><p className="text-3xl mb-2">🔍</p><p>{t('search.typeHint','Search across all chats, media and people')}</p></div>}
          {hasSearched && !loading && msgs.length===0 && media.length===0 && chats.length===0 && contacts.length===0 && <div className="text-center text-gray-500 py-8">{t('search.noResults','No results for "{{query}}"', {query})}</div>}
          {(filter==='all'||filter==='messages') && msgs.length>0 && (
            <div className="space-y-2">
              <div className="text-xs font-semibold text-gray-500">{msgs.length} {t('search.messages','messages')}</div>
              {msgs.map(m=>(
                <div key={m.id} onClick={()=>onSelectMessage?.(m)} className="p-3 rounded-xl border hover:bg-gray-50 cursor-pointer">
                  <div className="flex justify-between mb-1"><span className="text-sm font-semibold">{(m as any).sender?.displayName||t('searchMessages.unknown')}</span><span className="text-xs text-gray-400">{formatDistanceToNow(new Date(m.timestamp),{addSuffix:true})}</span></div>
                  <p className="text-sm text-gray-700 line-clamp-2">{hl(m.text)}</p>
                </div>
              ))}
            </div>
          )}
          {(filter==='all'||filter==='media') && media.length>0 && (
            <div className="space-y-2">
              <div className="text-xs font-semibold text-gray-500">{media.length} {t('search.media','media')}</div>
              {media.map(a=>(
                <div key={a.id} className="p-3 rounded-xl border-l-4 border-secondary bg-gray-50 flex gap-3">
                  <span className="material-symbols-outlined text-gray-500">{a.type==='image'?'image':a.type==='video'?'videocam':'description'}</span>
                  <div className="min-w-0"><div className="text-sm font-medium truncate">{a.fileName}</div><div className="text-xs text-gray-400">{a.type}</div></div>
                </div>
              ))}
            </div>
          )}
          {(filter==='all'||filter==='people') && (chats.length>0||contacts.length>0) && (
            <div className="space-y-2">
              <div className="text-xs font-semibold text-gray-500">{t('search.people','People & Chats')}</div>
              {chats.map((c:any)=>(<div key={c.id} className="p-2 rounded-lg bg-gray-50 flex items-center gap-2"><span className="material-symbols-outlined text-gray-400 text-[18px]">chat</span><span className="text-sm">{c.name||c.id.slice(0,8)}</span></div>))}
              {contacts.map((u:any)=>(<div key={u.id} className="p-2 rounded-lg bg-gray-50 flex items-center gap-2"><div className="w-7 h-7 rounded-full bg-primary text-white flex items-center justify-center text-xs">{u.displayName?.charAt(0)}</div><span className="text-sm">{u.displayName}</span><span className="text-xs text-gray-400">@{u.username}</span></div>))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

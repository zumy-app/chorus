import { useEffect, useState } from 'react'
import { createApiClient } from '@chorus/shared'
import AppHeader from '../components/AppHeader'
import BottomNav from '../components/BottomNav'

const client = createApiClient({
  baseURL: '/api/v1',
  storage: {
    getItem: async k => localStorage.getItem(k),
    setItem: async (k, v) => { localStorage.setItem(k, v) },
    removeItem: async k => { localStorage.removeItem(k) },
  },
})

const LANGS = ['en','es','fr','de','it','pt','ja','zh','ar','hi','ru']

export default function BecomeTeacher() {
  const [bio, setBio] = useState('')
  const [languages, setLanguages] = useState<string[]>([])
  const [expertise, setExpertise] = useState('')
  const [rate, setRate] = useState('20')
  const [videoUrl, setVideoUrl] = useState('')
  const [certs, setCerts] = useState<{type:string,issuer:string,year:number,fileUrl:string}[]>([])
  const [status, setStatus] = useState<string | null>(null)
  const [msg, setMsg] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    client.teacher.getMyApplication().then(a => {
      if (a) {
        setBio(a.bio)
        setLanguages(a.languages)
        setExpertise(a.expertise || '')
        setRate(String((a.rateCents)/100))
        setVideoUrl(a.videoUrl)
        setCerts((a.certificates||[]).map(c=>({type:c.type,issuer:c.issuer,year:c.year,fileUrl:c.fileUrl})))
        setStatus(a.status)
      }
    }).catch(()=>{})
  }, [])

  const toggleLang = (l:string) => setLanguages(prev=> prev.includes(l) ? prev.filter(x=>x!==l) : [...prev,l])

  const addCert = () => setCerts([...certs, {type:'language_certificate', issuer:'', year:new Date().getFullYear(), fileUrl:''}])

  const submit = async () => {
    setLoading(true); setMsg('')
    try {
      const rateCents = Math.round(parseFloat(rate)*100)
      const app = await client.teacher.apply({ bio, languages, expertise, rateCents, videoUrl, certificates: certs.map(c=>({type:c.type as any, issuer:c.issuer, year:c.year, fileUrl:c.fileUrl})) })
      setStatus(app.status); setMsg('Application submitted: '+app.status)
    } catch(e:any){ setMsg(e?.response?.data?.error || e.message) }
    setLoading(false)
  }

  return (
    <div className="h-screen flex flex-col bg-background">
      <AppHeader />
      <main className="flex-1 overflow-y-auto px-4 py-6 space-y-6 pb-32 max-w-md w-full mx-auto">
        <h2 className="text-xl font-bold">Become a Teacher</h2>
        {status && <p className="text-sm bg-primary-container p-2 rounded">Status: {status}</p>}
        {msg && <p className="text-sm text-center">{msg}</p>}

        <div className="space-y-4">
          <label className="block">
            <span className="text-sm font-medium">Bio (10-1000 chars)</span>
            <textarea value={bio} onChange={e=>setBio(e.target.value)} rows={4} className="w-full border rounded p-2 text-sm" placeholder="Tell students about yourself" />
            <span className="text-xs text-gray-500">{bio.length}/1000</span>
          </label>

          <div>
            <span className="text-sm font-medium">Languages you teach</span>
            <div className="flex flex-wrap gap-2 mt-1">
              {LANGS.map(l=>(
                <button key={l} onClick={()=>toggleLang(l)} className={`px-3 py-1 rounded-full text-sm border ${languages.includes(l) ? 'bg-primary text-white' : 'bg-surface'}`}>{l.toUpperCase()}</button>
              ))}
            </div>
          </div>

          <label className="block">
            <span className="text-sm font-medium">Expertise / specialties</span>
            <input value={expertise} onChange={e=>setExpertise(e.target.value)} className="w-full border rounded p-2 text-sm" placeholder="e.g. Conversational Spanish, DELE prep" />
          </label>

          <label className="block">
            <span className="text-sm font-medium">Hourly rate (USD)</span>
            <input type="number" min={1} value={rate} onChange={e=>setRate(e.target.value)} className="w-full border rounded p-2 text-sm" />
          </label>

          <label className="block">
            <span className="text-sm font-medium">Intro video URL (2-3 min demo, mp4/webm)</span>
            <input value={videoUrl} onChange={e=>setVideoUrl(e.target.value)} className="w-full border rounded p-2 text-sm" placeholder="https://..." />
          </label>

          <div className="border rounded p-3 space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium">Certificates</span>
              <button onClick={addCert} className="text-sm text-primary">+ Add</button>
            </div>
            {certs.map((c,i)=>(
              <div key={i} className="border rounded p-2 space-y-1">
                <div className="flex gap-2">
                  <select value={c.type} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, type:e.target.value}:x))} className="border rounded p-1 text-sm flex-1">
                    <option value="teaching_degree">Teaching degree</option>
                    <option value="language_certificate">Language cert</option>
                    <option value="other">Other</option>
                  </select>
                  <button onClick={()=>setCerts(certs.filter((_,j)=>j!==i))} className="text-xs text-error">Remove</button>
                </div>
                <input placeholder="Issuer" value={c.issuer} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, issuer:e.target.value}:x))} className="w-full border rounded p-1 text-sm" />
                <input placeholder="Year" type="number" value={c.year} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, year:parseInt(e.target.value)||0}:x))} className="w-full border rounded p-1 text-sm" />
                <input placeholder="File URL (PDF/JPG)" value={c.fileUrl} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, fileUrl:e.target.value}:x))} className="w-full border rounded p-1 text-sm" />
              </div>
            ))}
            {certs.length===0 && <p className="text-xs text-gray-500">Add at least one teaching or language certificate for approval.</p>}
          </div>

          <button onClick={submit} disabled={loading} className="w-full bg-primary text-white rounded-lg py-3 font-medium disabled:opacity-50">{loading ? 'Submitting...' : status ? 'Update application' : 'Submit application'}</button>
        </div>
      </main>
      <BottomNav />
    </div>
  )
}

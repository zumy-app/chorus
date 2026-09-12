import { useEffect, useState } from 'react'
import { createApiClient } from '@chorus/shared'
import { apiErrorMessage } from '@chorus/shared'
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
    } catch(e:any){
      // The API error envelope is {error:{kind,message}} — never stash the
      // raw object in state (React cannot render objects and unmounts).
      setMsg(apiErrorMessage(e))
    }
    setLoading(false)
  }

  const [step, setStep] = useState(1)

  return (
    <div className="min-h-screen flex flex-col bg-background text-on-background">
      <AppHeader />
      <main className="flex-grow flex flex-col md:flex-row w-full max-w-6xl mx-auto p-4 md:p-6 gap-6 pb-32">
        {/* Left Side: Bento Card Benefits */}
        <div className="w-full md:w-5/12 p-6 flex flex-col gap-4 bg-surface-container-low rounded-3xl shadow-sm">
          <div>
            <h1 className="font-headline-lg text-headline-lg text-on-surface mb-2 font-bold">Become a Chorus Tutor</h1>
            <p className="font-body-md text-body-md text-on-surface-variant">Join our global community of educators and turn your language skills into an engaging career.</p>
          </div>
          <div className="grid grid-cols-1 gap-3">
            <div className="bg-surface rounded-2xl p-4 shadow-sm border border-outline-variant/30 flex items-start gap-3">
              <div className="bg-secondary-fixed text-on-secondary-fixed p-2 rounded-xl shrink-0 mt-1">
                <span className="material-symbols-outlined">payments</span>
              </div>
              <div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface font-semibold mb-1">Earn on your terms</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant">Set your own hourly rate and manage your schedule flexibly.</p>
              </div>
            </div>
            <div className="bg-surface rounded-2xl p-4 shadow-sm border border-outline-variant/30 flex items-start gap-3">
              <div className="bg-tertiary-fixed text-on-tertiary-fixed p-2 rounded-xl shrink-0 mt-1">
                <span className="material-symbols-outlined">public</span>
              </div>
              <div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface font-semibold mb-1">Reach global learners</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant">Connect with motivated students from all over the world.</p>
              </div>
            </div>
            <div className="bg-surface rounded-2xl p-4 shadow-sm border border-outline-variant/30 flex items-start gap-3 relative overflow-hidden">
              <div className="bg-primary-fixed text-on-primary-fixed p-2 rounded-xl shrink-0 mt-1">
                <span className="material-symbols-outlined">forum</span>
              </div>
              <div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface font-semibold mb-1">Integrated chat & video</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant">Teach seamlessly using our purpose-built virtual classrooms.</p>
              </div>
            </div>
          </div>
        </div>

        {/* Right Side: Step Form */}
        <div className="w-full md:w-7/12 p-6 flex flex-col bg-surface rounded-3xl shadow-sm border border-outline-variant/20">
          {/* Progress Tracker */}
          <div className="flex items-center justify-between mb-6 pb-4 border-b border-outline-variant/20">
            <div className="flex items-center w-full gap-2">
              <button onClick={() => setStep(1)} className="flex flex-col items-center flex-1">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 1 ? 'bg-primary text-on-primary' : 'bg-surface-variant text-on-surface-variant'}`}>1</div>
                <span className="text-xs mt-1 font-medium">Basic Info</span>
              </button>
              <div className="h-1 flex-1 bg-outline-variant/40 rounded-full"><div className={`h-full bg-primary rounded-full transition-all ${step > 1 ? 'w-full' : 'w-0'}`} /></div>
              <button onClick={() => setStep(2)} className="flex flex-col items-center flex-1">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 2 ? 'bg-primary text-on-primary' : 'bg-surface-variant text-on-surface-variant'}`}>2</div>
                <span className="text-xs mt-1 font-medium">Expertise</span>
              </button>
              <div className="h-1 flex-1 bg-outline-variant/40 rounded-full"><div className={`h-full bg-primary rounded-full transition-all ${step > 2 ? 'w-full' : 'w-0'}`} /></div>
              <button onClick={() => setStep(3)} className="flex flex-col items-center flex-1">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 3 ? 'bg-primary text-on-primary' : 'bg-surface-variant text-on-surface-variant'}`}>3</div>
                <span className="text-xs mt-1 font-medium">Profile & Rates</span>
              </button>
            </div>
          </div>

          {status && <div className="mb-4 bg-primary-container/20 text-primary p-3 rounded-xl text-sm font-medium">Application Status: {status}</div>}
          {msg && <div className="mb-4 bg-surface-container p-3 rounded-xl text-sm text-center font-medium">{msg}</div>}

          {step === 1 && (
            <div className="space-y-4 flex-1">
              <h2 className="text-xl font-bold text-on-surface">Step 1: Tell us about yourself</h2>
              <p className="text-sm text-on-surface-variant">Introduce yourself to future students.</p>
              <label className="block">
                <span className="text-sm font-medium text-on-surface">Short Bio (10-1000 characters)</span>
                <textarea value={bio} onChange={e=>setBio(e.target.value)} rows={4} className="w-full rounded-xl border border-outline-variant/50 p-3 text-sm mt-1 focus:ring-2 focus:ring-primary outline-none" placeholder="Write a brief introduction about your teaching style, experience, and background..." />
                <span className="text-xs text-outline mt-1 block text-right">{bio.length}/1000</span>
              </label>
              <label className="block">
                <span className="text-sm font-medium text-on-surface">Specialties & Expertise</span>
                <input value={expertise} onChange={e=>setExpertise(e.target.value)} className="w-full rounded-xl border border-outline-variant/50 p-3 text-sm mt-1 focus:ring-2 focus:ring-primary outline-none" placeholder="e.g. Conversational Spanish, DELE prep, Business Spanish" />
              </label>
            </div>
          )}

          {step === 2 && (
            <div className="space-y-4 flex-1">
              <h2 className="text-xl font-bold text-on-surface">Step 2: Languages & Qualifications</h2>
              <p className="text-sm text-on-surface-variant">Select the languages you teach and add your certificates.</p>
              <div>
                <span className="text-sm font-medium text-on-surface block mb-2">Languages you teach</span>
                <div className="flex flex-wrap gap-2">
                  {LANGS.map(l=>(
                    <button key={l} type="button" onClick={()=>toggleLang(l)} className={`px-4 py-2 rounded-xl text-sm font-medium border transition-colors ${languages.includes(l) ? 'bg-primary text-on-primary border-primary' : 'bg-surface-bright text-on-surface border-outline-variant/50'}`}>{l.toUpperCase()}</button>
                  ))}
                </div>
              </div>
              <div className="border border-outline-variant/50 rounded-2xl p-4 space-y-3 bg-surface-bright">
                <div className="flex justify-between items-center">
                  <span className="text-sm font-semibold text-on-surface">Certifications & Degrees</span>
                  <button type="button" onClick={addCert} className="text-sm text-primary font-medium">+ Add Certificate</button>
                </div>
                {certs.map((c,i)=>(
                  <div key={i} className="border border-outline-variant/30 rounded-xl p-3 space-y-2 bg-surface">
                    <div className="flex gap-2">
                      <select value={c.type} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, type:e.target.value}:x))} className="border rounded-lg p-2 text-sm flex-1 bg-surface-bright">
                        <option value="teaching_degree">Teaching Degree</option>
                        <option value="language_certificate">Language Cert (TEFL/CELTA/DELE)</option>
                        <option value="other">Other Qualification</option>
                      </select>
                      <button type="button" onClick={()=>setCerts(certs.filter((_,j)=>j!==i))} className="text-xs text-error font-medium px-2">Remove</button>
                    </div>
                    <input placeholder="Issuer / Organization" value={c.issuer} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, issuer:e.target.value}:x))} className="w-full border rounded-lg p-2 text-sm bg-surface-bright" />
                    <div className="flex gap-2">
                      <input placeholder="Year" type="number" value={c.year} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, year:parseInt(e.target.value)||0}:x))} className="w-1/3 border rounded-lg p-2 text-sm bg-surface-bright" />
                      <input placeholder="Credential / File URL" value={c.fileUrl} onChange={e=>setCerts(certs.map((x,j)=> j===i? {...x, fileUrl:e.target.value}:x))} className="w-2/3 border rounded-lg p-2 text-sm bg-surface-bright" />
                    </div>
                  </div>
                ))}
                {certs.length===0 && <p className="text-xs text-on-surface-variant">Add at least one teaching or language certificate for verification.</p>}
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4 flex-1">
              <h2 className="text-xl font-bold text-on-surface">Step 3: Intro Video & Pricing</h2>
              <p className="text-sm text-on-surface-variant">Set your hourly rate and intro video link.</p>
              <div className="bg-surface-bright rounded-2xl p-4 border border-outline-variant/50">
                <label className="block text-sm font-semibold text-on-surface mb-2">Hourly Rate (USD)</label>
                <div className="flex items-center gap-3">
                  <span className="text-xl font-bold text-on-surface">$</span>
                  <input type="number" min={1} max={150} value={rate} onChange={e=>setRate(e.target.value)} className="w-32 border rounded-xl p-3 text-lg font-bold text-center bg-surface" />
                  <span className="text-sm text-on-surface-variant">USD / hour</span>
                </div>
              </div>
              <label className="block">
                <span className="text-sm font-medium text-on-surface">Intro Video URL (mp4, webm or demo link)</span>
                <input value={videoUrl} onChange={e=>setVideoUrl(e.target.value)} className="w-full border rounded-xl p-3 text-sm mt-1 focus:ring-2 focus:ring-primary outline-none" placeholder="https://example.com/intro-video.mp4" />
              </label>
            </div>
          )}

          {/* Navigation Actions */}
          <div className="mt-8 pt-4 border-t border-outline-variant/20 flex justify-between items-center">
            {step > 1 ? (
              <button type="button" onClick={() => setStep(step - 1)} className="px-4 py-2 text-sm font-medium text-on-surface-variant hover:text-on-surface">Back</button>
            ) : <div />}
            {step < 3 ? (
              <button type="button" onClick={() => setStep(step + 1)} className="bg-primary text-on-primary font-medium px-6 py-3 rounded-xl text-sm hover:bg-primary-container transition-colors">Continue →</button>
            ) : (
              <button type="button" onClick={submit} disabled={loading} className="bg-primary text-on-primary font-medium px-8 py-3 rounded-xl text-sm hover:bg-primary-container disabled:opacity-50 transition-colors">{loading ? 'Submitting...' : status ? 'Update Application' : 'Submit Application'}</button>
            )}
          </div>
        </div>
      </main>
      <BottomNav />
    </div>
  )
}

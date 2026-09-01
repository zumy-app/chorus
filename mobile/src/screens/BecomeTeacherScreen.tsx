import React, { useEffect, useState } from 'react'
import { View, Text, TextInput, TouchableOpacity, ScrollView, Alert } from 'react-native'
import { api } from '../services/api'

const LANGS = ['en','es','fr','de','it','pt','ja','zh','ar','hi','ru']

export default function BecomeTeacherScreen() {
  const [bio, setBio] = useState('')
  const [languages, setLanguages] = useState<string[]>([])
  const [expertise, setExpertise] = useState('')
  const [rate, setRate] = useState('20')
  const [videoUrl, setVideoUrl] = useState('')
  const [certs, setCerts] = useState<any[]>([])
  const [status, setStatus] = useState<string|null>(null)

  useEffect(()=>{ api.get('/teachers/me').then((r:any)=>{ const a=r.data.application; if(a){ setBio(a.bio); setLanguages(a.languages); setExpertise(a.expertise||''); setRate(String(a.rateCents/100)); setVideoUrl(a.videoUrl); setCerts((a.certificates||[]).map((c:any)=>({type:c.type,issuer:c.issuer,year:String(c.year),fileUrl:c.fileUrl}))); setStatus(a.status)}}).catch(()=>{}) },[])

  const toggle = (l:string)=> setLanguages(p=> p.includes(l)? p.filter(x=>x!==l): [...p,l])

  const submit = async()=>{
    try{
      const rateCents=Math.round(parseFloat(rate)*100)
      const payload={bio,languages,expertise,rateCents,videoUrl,certificates:certs.map(c=>({type:c.type,issuer:c.issuer,year:parseInt(c.year)||0,fileUrl:c.fileUrl}))}
      const r:any = await api.post('/teachers/apply', payload)
      setStatus(r.data.application.status); Alert.alert('Submitted', r.data.application.status)
    }catch(e:any){ Alert.alert('Error', e?.response?.data?.error || e.message)}
  }

  return (
    <ScrollView contentContainerStyle={{padding:16, gap:16}}>
      <Text style={{fontSize:20,fontWeight:'700'}}>Become a Teacher</Text>
      {status && <Text>Status: {status}</Text>}
      <Text>Bio (10-1000)</Text>
      <TextInput value={bio} onChangeText={setBio} multiline style={{borderWidth:1,borderRadius:8,padding:8,minHeight:80}} placeholder="Tell students about yourself" />
      <Text>Languages you teach</Text>
      <View style={{flexDirection:'row', flexWrap:'wrap', gap:8}}>
        {LANGS.map(l=>(
          <TouchableOpacity key={l} onPress={()=>toggle(l)} style={{paddingHorizontal:12,paddingVertical:6,borderRadius:16,borderWidth:1,backgroundColor: languages.includes(l)? '#6366F1':'white'}}>
            <Text style={{color: languages.includes(l)? 'white':'black'}}>{l.toUpperCase()}</Text>
          </TouchableOpacity>
        ))}
      </View>
      <Text>Expertise</Text>
      <TextInput value={expertise} onChangeText={setExpertise} style={{borderWidth:1,borderRadius:8,padding:8}} placeholder="e.g. Conversational Spanish" />
      <Text>Hourly rate USD</Text>
      <TextInput value={rate} onChangeText={setRate} keyboardType="numeric" style={{borderWidth:1,borderRadius:8,padding:8}} />
      <Text>Intro video URL</Text>
      <TextInput value={videoUrl} onChangeText={setVideoUrl} style={{borderWidth:1,borderRadius:8,padding:8}} placeholder="https://..." />
      <View style={{flexDirection:'row', justifyContent:'space-between', alignItems:'center'}}>
        <Text>Certificates</Text>
        <TouchableOpacity onPress={()=>setCerts([...certs,{type:'language_certificate',issuer:'',year:String(new Date().getFullYear()),fileUrl:''}])}><Text style={{color:'#6366F1'}}>+ Add</Text></TouchableOpacity>
      </View>
      {certs.map((c,i)=>(
        <View key={i} style={{borderWidth:1,borderRadius:8,padding:8,gap:6}}>
          <View style={{flexDirection:'row', gap:8}}>
            <TextInput value={c.type} onChangeText={v=>setCerts(certs.map((x,j)=>j===i?{...x,type:v}:x))} style={{flex:1,borderWidth:1,borderRadius:6,padding:6}} placeholder="type" />
            <TouchableOpacity onPress={()=>setCerts(certs.filter((_,j)=>j!==i))}><Text style={{color:'red'}}>Remove</Text></TouchableOpacity>
          </View>
          <TextInput value={c.issuer} onChangeText={v=>setCerts(certs.map((x,j)=>j===i?{...x,issuer:v}:x))} style={{borderWidth:1,borderRadius:6,padding:6}} placeholder="Issuer" />
          <TextInput value={c.year} onChangeText={v=>setCerts(certs.map((x,j)=>j===i?{...x,year:v}:x))} style={{borderWidth:1,borderRadius:6,padding:6}} placeholder="Year" keyboardType="numeric" />
          <TextInput value={c.fileUrl} onChangeText={v=>setCerts(certs.map((x,j)=>j===i?{...x,fileUrl:v}:x))} style={{borderWidth:1,borderRadius:6,padding:6}} placeholder="File URL" />
        </View>
      ))}
      <TouchableOpacity onPress={submit} style={{backgroundColor:'#6366F1',padding:14,borderRadius:12,alignItems:'center'}}><Text style={{color:'white',fontWeight:'600'}}>{status?'Update application':'Submit application'}</Text></TouchableOpacity>
    </ScrollView>
  )
}

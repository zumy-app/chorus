import React, { useState, useCallback } from 'react';
import { View, Text, TextInput, TouchableOpacity, ScrollView, StyleSheet, ActivityIndicator } from 'react-native';
import { createApiClient, resolveApiConfig } from '@chorus/shared';
import storage from '../utils/storage';
import { COLOR, FONTS } from '../theme';

const { baseURL } = resolveApiConfig({ platform: 'ios' as any, dev: __DEV__, origin: process.env.EXPO_PUBLIC_API_URL || '', version: process.env.EXPO_PUBLIC_API_VERSION });
const client = createApiClient({ baseURL, storage: storage as any });
type Filter='all'|'messages'|'media'|'people'

export default function UniversalSearchScreen({ navigation }: any){
  const [q,setQ]=useState('')
  const [filter,setFilter]=useState<Filter>('all')
  const [msgs,setMsgs]=useState<any[]>([])
  const [media,setMedia]=useState<any[]>([])
  const [chats,setChats]=useState<any[]>([])
  const [contacts,setContacts]=useState<any[]>([])
  const [loading,setLoading]=useState(false)
  const [hasSearched,setHasSearched]=useState(false)
  const [recent,setRecent]=useState<string[]>([])
  React.useEffect(()=>{ storage.getItem('chorus_recent_searches').then(v=>{ try{ if(v) setRecent(JSON.parse(v))}catch{}}) },[])
  const saveRecent=useCallback(async(query:string)=>{
    if(!query.trim())return
    const next=[query, ...recent.filter(x=>x!==query)].slice(0,8)
    setRecent(next); await storage.setItem('chorus_recent_searches', JSON.stringify(next))
  },[recent])
  const doSearch=useCallback(async(query?:string)=>{
    const term=(query??q).trim()
    if(!term) return
    setLoading(true); setHasSearched(true); saveRecent(term)
    try{
      const [r1,r2,r3,r4]=await Promise.allSettled([
        (filter==='all'||filter==='messages') ? client.search.universal(term) : Promise.resolve({messages:[],media:[],total:0,mediaTotal:0,hasMore:false} as any),
        (filter==='all'||filter==='media') ? client.search.media(term) : Promise.resolve({media:[],total:0,hasMore:false} as any),
        (filter==='all'||filter==='people') ? client.search.chats(term) : Promise.resolve([] as any),
        (filter==='all'||filter==='people') ? client.search.contacts(term) : Promise.resolve([] as any),
      ])
      if(r1.status==='fulfilled'){ const r:any=r1.value; setMsgs(r.messages||[]); if(filter==='all') setMedia(r.media||[]) }
      if(r2.status==='fulfilled' && filter!=='all') setMedia((r2.value as any).media||[])
      if(r3.status==='fulfilled') setChats(r3.value as any)
      if(r4.status==='fulfilled') setContacts(r4.value as any)
    }catch{} finally{ setLoading(false)}
  },[q,filter,saveRecent])
  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={()=>navigation.goBack()} style={styles.back}><Text style={styles.backText}>‹</Text></TouchableOpacity>
        <View style={styles.inputWrap}>
          <Text style={styles.searchIcon}>🔍</Text>
          <TextInput value={q} onChangeText={setQ} onSubmitEditing={()=>doSearch()} placeholder="Search messages, media, or people..." placeholderTextColor={COLOR.onSurfaceVariant} style={styles.input} returnKeyType="search" autoFocus />
          {q.length>0 && <TouchableOpacity onPress={()=>{setQ(''); setMsgs([]); setMedia([]); setChats([]); setContacts([]); setHasSearched(false)}}><Text style={styles.clear}>✕</Text></TouchableOpacity>}
        </View>
      </View>
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filters}>
        {(['all','messages','media','people'] as Filter[]).map(f=>(
          <TouchableOpacity key={f} onPress={()=>setFilter(f)} style={[styles.chip, filter===f && styles.chipActive]}>
            <Text style={[styles.chipText, filter===f && styles.chipTextActive]}>{f}</Text>
          </TouchableOpacity>
        ))}
      </ScrollView>
      <ScrollView style={styles.body} contentContainerStyle={{padding:16, gap:16}}>
        {!hasSearched && recent.length>0 && (
          <View>
            <View style={{flexDirection:'row', justifyContent:'space-between', marginBottom:8}}><Text style={styles.sectionTitle}>Recent Searches</Text><TouchableOpacity onPress={async()=>{setRecent([]); await storage.removeItem('chorus_recent_searches')}}><Text style={{color:COLOR.primary, fontSize:12}}>Clear</Text></TouchableOpacity></View>
            <View style={{flexDirection:'row', flexWrap:'wrap', gap:8}}>{recent.map(r=>(<TouchableOpacity key={r} onPress={()=>{setQ(r); doSearch(r)}} style={styles.recentPill}><Text style={styles.recentText}>🕘 {r}</Text></TouchableOpacity>))}</View>
          </View>
        )}
        {loading && <ActivityIndicator color={COLOR.primary} />}
        {!loading && hasSearched && msgs.length===0 && media.length===0 && chats.length===0 && contacts.length===0 && <View style={{alignItems:'center', paddingTop:40}}><Text style={{fontSize:32}}>📭</Text><Text style={{color:COLOR.onSurfaceVariant, marginTop:8}}>No results for "{q}"</Text></View>}
        {(filter==='all'||filter==='messages') && msgs.length>0 && (
          <View style={{gap:8}}>
            <Text style={styles.sectionTitle}>Messages — {msgs.length}</Text>
            {msgs.map((m:any)=>(
              <View key={m.id} style={styles.card}>
                <View style={styles.avatar}><Text style={{fontWeight:'700'}}>{(m.sender?.displayName||'?').charAt(0)}</Text></View>
                <View style={{flex:1}}>
                  <View style={{flexDirection:'row', justifyContent:'space-between'}}><Text style={styles.cardTitle}>{m.sender?.displayName||m.senderId?.slice(0,8)}</Text><Text style={styles.cardDate}>{m.timestamp? new Date(m.timestamp).toLocaleDateString():''}</Text></View>
                  <Text style={styles.cardBody} numberOfLines={2}>{m.text}</Text>
                </View>
              </View>
            ))}
          </View>
        )}
        {(filter==='all'||filter==='media') && media.length>0 && (
          <View style={{gap:8}}>
            <Text style={styles.sectionTitle}>Media — {media.length}</Text>
            {media.map((a:any)=>(
              <View key={a.id} style={[styles.card,{borderLeftWidth:4, borderLeftColor:COLOR.secondary}]}>
                <Text style={{fontSize:18}}>{a.type==='image'?'🖼️':a.type==='video'?'🎬':a.type==='audio'?'🎧':'📄'}</Text>
                <View style={{flex:1}}><Text style={styles.cardTitle}>{a.fileName||a.type}</Text><Text style={styles.cardBody}>{a.type}</Text></View>
              </View>
            ))}
          </View>
        )}
        {(filter==='all'||filter==='people') && (chats.length>0||contacts.length>0) && (
          <View style={{gap:8}}>
            <Text style={styles.sectionTitle}>People & Chats</Text>
            {chats.map((c:any)=>(<View key={c.id} style={styles.card}><Text>💬</Text><Text style={styles.cardTitle}>{c.name||c.id.slice(0,8)}</Text></View>))}
            {contacts.map((u:any)=>(<View key={u.id} style={styles.card}><View style={[styles.avatar,{backgroundColor:COLOR.primaryContainer}]}><Text>{u.displayName?.charAt(0)}</Text></View><View><Text style={styles.cardTitle}>{u.displayName}</Text><Text style={styles.cardBody}>@{u.username}</Text></View></View>))}
          </View>
        )}
      </ScrollView>
    </View>
  )
}
const styles=StyleSheet.create({
  container:{flex:1, backgroundColor:COLOR.background},
  header:{flexDirection:'row', alignItems:'center', padding:12, gap:8, backgroundColor:COLOR.surface, borderBottomWidth:1, borderBottomColor:COLOR.outlineVariant},
  back:{width:32, height:32, alignItems:'center', justifyContent:'center'},
  backText:{fontSize:24, color:COLOR.onSurface},
  inputWrap:{flex:1, flexDirection:'row', alignItems:'center', backgroundColor:COLOR.surfaceContainerLow, borderRadius:20, paddingHorizontal:12, height:40, gap:6},
  searchIcon:{fontSize:16},
  input:{flex:1, fontSize:14, color:COLOR.onSurface},
  clear:{fontSize:16, color:COLOR.onSurfaceVariant, padding:4},
  filters:{paddingHorizontal:12, paddingVertical:8, gap:8},
  chip:{paddingHorizontal:14, paddingVertical:6, borderRadius:16, backgroundColor:COLOR.surfaceContainerHigh},
  chipActive:{backgroundColor:COLOR.primary},
  chipText:{fontSize:12, fontWeight:'600', color:COLOR.onSurfaceVariant, textTransform:'capitalize'},
  chipTextActive:{color:'#fff'},
  body:{flex:1},
  sectionTitle:{fontSize:12, fontWeight:'700', color:COLOR.onSurfaceVariant, textTransform:'uppercase', letterSpacing:0.5},
  card:{flexDirection:'row', gap:12, backgroundColor:COLOR.surfaceContainerLowest, borderRadius:12, padding:12, alignItems:'center', shadowColor:'#000', shadowOpacity:0.05, shadowRadius:4, elevation:1},
  avatar:{width:40, height:40, borderRadius:20, backgroundColor:COLOR.tertiaryContainer, alignItems:'center', justifyContent:'center'},
  cardTitle:{fontSize:14, fontWeight:'600', color:COLOR.onSurface},
  cardDate:{fontSize:11, color:COLOR.onSurfaceVariant},
  cardBody:{fontSize:13, color:COLOR.onSurfaceVariant, marginTop:2},
  recentPill:{paddingHorizontal:10, paddingVertical:6, backgroundColor:COLOR.surfaceContainerLow, borderRadius:8, borderWidth:1, borderColor:COLOR.outlineVariant},
  recentText:{fontSize:13, color:COLOR.onSurface},
})

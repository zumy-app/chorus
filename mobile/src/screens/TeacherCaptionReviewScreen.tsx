import React, { useEffect, useState } from 'react';
import { View, Text, FlatList, TouchableOpacity, TextInput, ScrollView } from 'react-native';
import apiService from '../services/api';
import { api } from '../services/api';

export default function TeacherCaptionReviewScreen() {
  const [items, setItems] = useState<any[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [sel, setSel] = useState<any>(null);
  const [reviews, setReviews] = useState<any[]>([]);
  const [rating, setRating] = useState(5);
  const [corrected, setCorrected] = useState('');
  const [feedback, setFeedback] = useState('');
  const [msg, setMsg] = useState('');

  const load = async () => {
    try {
      const r: any = await (api as any).get('/captions/review-queue?limit=20');
      setItems(r.data.items || []);
      const s: any = await (api as any).get('/captions/quality-stats');
      setStats(s.data);
    } catch {}
  };
  useEffect(()=>{load()},[]);
  const open = async (it:any) => {
    setSel(it); setCorrected(Object.values(it.translations||{})[0] as string || '');
    try { const r:any = await (api as any).get(`/calls/${it.callId}/captions/${it.segmentIndex}/reviews`); setReviews(r.data.reviews||[]);} catch {}
  };
  const submit = async () => {
    if(!sel) return;
    try { await (api as any).post(`/calls/${sel.callId}/captions/${sel.segmentIndex}/review`, { rating, correctedText: corrected, feedback }); setMsg('Submitted'); load(); const r:any=await (api as any).get(`/calls/${sel.callId}/captions/${sel.segmentIndex}/reviews`); setReviews(r.data.reviews||[]);} catch(e:any){setMsg(e?.response?.data?.error||'Failed');}
  };
  return (
    <ScrollView style={{flex:1,padding:12}}>
      <Text style={{fontSize:18,fontWeight:'700'}}>Caption Review</Text>
      {stats && <Text style={{fontSize:12,color:'#666'}}>Total:{stats.totalCaptions} Reviewed:{stats.reviewedCount} Avg:{Number(stats.avgRating||0).toFixed(2)}</Text>}
      <Text style={{marginTop:8,fontWeight:'600'}}>Queue</Text>
      {items.map((it,i)=>(<TouchableOpacity key={i} onPress={()=>open(it)} style={{borderWidth:1,borderColor: sel?.callId===it.callId&&sel?.segmentIndex===it.segmentIndex?'blue':'#ddd',padding:8,borderRadius:8,marginTop:6}}><Text numberOfLines={1}>{it.originalText}</Text><Text style={{fontSize:11,color:'#888'}}>{it.originalLanguage} → {Object.keys(it.translations||{}).join(',')}</Text></TouchableOpacity>))}
      {sel && <View style={{marginTop:12,borderWidth:1,borderColor:'#ddd',padding:10,borderRadius:8}}>
        <Text>Original: {sel.originalText}</Text>
        <View style={{flexDirection:'row',marginTop:8}}>{[1,2,3,4,5].map(n=>(<TouchableOpacity key={n} onPress={()=>setRating(n)} style={{width:32,height:32,backgroundColor:rating>=n?'#facc15':'#eee',alignItems:'center',justifyContent:'center',marginRight:4,borderRadius:4}}><Text>{n}</Text></TouchableOpacity>))}</View>
        <TextInput value={corrected} onChangeText={setCorrected} placeholder="Corrected" style={{borderWidth:1,borderColor:'#ddd',borderRadius:6,padding:6,marginTop:8}} />
        <TextInput value={feedback} onChangeText={setFeedback} placeholder="Feedback" style={{borderWidth:1,borderColor:'#ddd',borderRadius:6,padding:6,marginTop:8}} />
        <TouchableOpacity onPress={submit} style={{backgroundColor:'#2563eb',padding:10,borderRadius:6,marginTop:8,alignItems:'center'}}><Text style={{color:'#fff'}}>Submit Review</Text></TouchableOpacity>
        {msg? <Text style={{color:'green',marginTop:4}}>{msg}</Text>:null}
        <Text style={{marginTop:8,fontWeight:'600',fontSize:12}}>Previous ({reviews.length})</Text>
        {reviews.map((r:any)=>(<Text key={r.id} style={{fontSize:12,borderBottomWidth:1,borderColor:'#eee',paddingVertical:4}}>★{r.rating} {r.correctedText||r.translatedText} {r.feedback}</Text>))}
      </View>}
    </ScrollView>
  );
}

import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TextInput, Pressable, Alert } from 'react-native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';

export default function PayoutsScreen() {
  const [overview,setOverview]=useState<any>(null);
  const [methods,setMethods]=useState<any[]>([]);
  const [history,setHistory]=useState<any[]>([]);
  const [loading,setLoading]=useState(true);
  const [label,setLabel]=useState('');
  const [details,setDetails]=useState('');
  const [amt,setAmt]=useState('10');

  const load=async()=>{ setLoading(true); try{ const [ov,ms,hs]=await Promise.all([(apiService as any).getPayoutOverview(),(apiService as any).getPayoutMethods(),(apiService as any).getPayoutHistory({limit:10})]); setOverview(ov.overview??ov); setMethods(ms.methods??ms); setHistory((hs.payouts??hs)??[]);} catch{} setLoading(false);};
  useEffect(()=>{load();},[]);

  if(loading) return <View style={styles.center}><ActivityIndicator color={COLOR.primary}/></View>;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {overview && <View style={styles.card}><Text style={styles.cardTitle}>Earnings</Text><Text style={styles.sub}>Total Lifetime Earnings</Text><Text style={styles.hero}>${((overview.availableCents??0)/100).toFixed(2)} available</Text><Text style={styles.sub}>Pending ${(overview.pendingCents??0)/100} · Fee {overview.platformFeePct??10}%</Text></View>}
      {overview && <View style={styles.card}><Text style={styles.cardTitle}>Breakdown</Text><View style={styles.row}><Text style={styles.body}>Gross</Text><Text style={styles.body}>${((overview.totalGrossCents??overview.lifetimeGross??0)/100).toFixed(2)}</Text></View><View style={styles.row}><Text style={styles.body}>Platform Fee {overview.platformFeePct??10}%</Text><Text style={styles.body}>-</Text></View><View style={styles.row}><Text style={styles.body}>Net</Text><Text style={styles.body}>${((overview.totalNetCents??overview.lifetimeNet??0)/100).toFixed(2)}</Text></View></View>}
      <View style={styles.card}><Text style={styles.cardTitle}>Payout Methods</Text>{methods.length===0?<Text style={styles.sub}>No methods</Text>:methods.map((m:any)=><View key={m.id} style={styles.row}><Text style={styles.body}>{m.type} · {m.label}</Text><Pressable onPress={async()=>{await (apiService as any).removePayoutMethod(m.id); load();}}><Text style={styles.danger}>Remove</Text></Pressable></View>)}
        <View style={styles.inputRow}><TextInput value={label} onChangeText={setLabel} placeholder="Label" style={styles.inputSmall} placeholderTextColor={COLOR.outline}/><TextInput value={details} onChangeText={setDetails} placeholder="PayPal email / IBAN" style={styles.inputSmall} placeholderTextColor={COLOR.outline}/><Pressable style={styles.addBtn} onPress={async()=>{try{await (apiService as any).addPayoutMethod({type:'paypal',label:label||'paypal',details:details||label}); setLabel('');setDetails(''); load();}catch(e:any){Alert.alert('Error',e?.response?.data?.error||e.message)}} }><Text style={styles.addBtnText}>Add</Text></Pressable></View>
      </View>
      <View style={styles.card}><Text style={styles.cardTitle}>Withdraw</Text><View style={styles.inputRow}><TextInput value={amt} onChangeText={setAmt} keyboardType="numeric" style={styles.input} placeholderTextColor={COLOR.outline}/><Pressable style={styles.primaryBtn} onPress={async()=>{try{await (apiService as any).requestPayout({amountCents:Math.round(parseFloat(amt)*100)}); Alert.alert('Success','Payout requested'); load();}catch(e:any){Alert.alert('Failed',e?.response?.data?.error||e.message)}} }><Text style={styles.primaryBtnText}>Withdraw</Text></Pressable></View></View>
      <View style={styles.card}><Text style={styles.cardTitle}>History</Text>{history.length===0?<Text style={styles.sub}>No payouts</Text>:history.map((p:any)=><View key={p.id} style={styles.row}><Text style={styles.body}>${(p.amountCents/100).toFixed(2)} · {p.status}</Text><Text style={styles.sub}>{new Date(p.createdAt).toLocaleDateString()}</Text></View>)}</View>
    </ScrollView>
  );
}
const styles=StyleSheet.create({
  container:{flex:1,backgroundColor:COLOR.background},
  content:{padding:SPACING.marginMobile,gap:SPACING.stackMd,paddingBottom:32},
  center:{flex:1,backgroundColor:COLOR.background,alignItems:'center',justifyContent:'center',padding:24},
  card:{backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:SPACING.stackMd,gap:8,...SHADOWS.elevation1},
  cardTitle:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:16},
  hero:{...TYPOGRAPHY.headlineMd,color:COLOR.primary,fontFamily:FONTS.headline,fontSize:22},
  body:{...TYPOGRAPHY.bodySm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body},
  sub:{...TYPOGRAPHY.labelSm,color:COLOR.outline,fontFamily:FONTS.label},
  danger:{...TYPOGRAPHY.labelSm,color:COLOR.error,fontFamily:FONTS.label},
  row:{flexDirection:'row',justifyContent:'space-between',alignItems:'center',borderTopWidth:1,borderTopColor:COLOR.outlineVariant,paddingTop:8},
  input:{flex:1,borderWidth:1,borderColor:COLOR.outlineVariant,borderRadius:RADIUS.full,paddingHorizontal:12,height:40,color:COLOR.onSurface,fontFamily:FONTS.body},
  inputSmall:{flex:1,borderWidth:1,borderColor:COLOR.outlineVariant,borderRadius:8,paddingHorizontal:8,height:36,color:COLOR.onSurface,fontSize:12},
  inputRow:{flexDirection:'row',gap:8,alignItems:'center'},
  primaryBtn:{backgroundColor:COLOR.primary,borderRadius:RADIUS.full,paddingHorizontal:16,paddingVertical:10},
  primaryBtnText:{...TYPOGRAPHY.labelMd,color:COLOR.onPrimary,fontFamily:FONTS.label},
  addBtn:{backgroundColor:COLOR.primary,borderRadius:8,paddingHorizontal:12,paddingVertical:8},
  addBtnText:{color:COLOR.onPrimary,fontFamily:FONTS.label,fontSize:12},
});

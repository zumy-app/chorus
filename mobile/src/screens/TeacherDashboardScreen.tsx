import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, Pressable } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';

export default function TeacherDashboardScreen() {
  const nav = useNavigation<any>();
  const [dash, setDash] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(()=>{ let a=true; (async()=>{ try{ const r = await (apiService as any).getTeacherDashboard(); if(a) setDash(r.dashboard ?? r);} catch(e:any){ if(a) setErr(e?.response?.data?.error||e.message);} finally{ if(a) setLoading(false);} })(); return()=>{a=false};},[]);

  if(loading) return <View style={styles.center}><ActivityIndicator color={COLOR.primary}/></View>;
  if(err) return <View style={styles.center}><Text style={styles.body}>{err}</Text><Pressable onPress={()=>nav.navigate('BecomeTeacher')}><Text style={styles.link}>Become a teacher</Text></Pressable></View>;

  const pct = dash?.checklist?.completionPct ?? dash?.checklist?.percent ?? 0;
  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Teacher Dashboard</Text>
      <View style={styles.welcome}><Text style={styles.welcomeTitle}>Welcome back!</Text><Text style={styles.welcomeSub}>Here&apos;s what&apos;s happening today.</Text></View>
      {dash?.earnings && <View style={styles.card}><Text style={styles.cardTitle}>Earnings Overview</Text><View style={styles.statsRow}><View style={styles.stat}><Text style={styles.statLabel}>Total Earned</Text><Text style={styles.statValue}>${((dash.earnings.totalGrossCents??0)/100).toFixed(2)}</Text></View><View style={styles.stat}><Text style={styles.statLabel}>Pending</Text><Text style={[styles.statValue,{color:COLOR.secondary}]}>${((dash.earnings.pendingCents??dash.earnings.pendingGrossCents??0)/100).toFixed(2)}</Text></View><View style={styles.stat}><Text style={styles.statLabel}>Fee {dash.earnings.platformFeePct??15}%</Text><Text style={styles.statValue}>-</Text></View></View></View>}
      <View style={styles.cardDark}><Text style={styles.cardTitleLight}>Premium Program</Text><Text style={styles.bodyLight}>You are enrolled in premium sessions.</Text><Pressable style={styles.lightBtn} onPress={()=>nav.navigate('Payouts')}><Text style={styles.lightBtnText}>Manage Premium Settings</Text></Pressable></View>
      {dash?.checklist && <View style={styles.card}><Text style={styles.cardTitle}>Profile Completion — {pct}%</Text><View style={styles.track}><View style={[styles.fill,{width:`${pct}%`}]} /></View><Text style={styles.sub}>Complete your profile to attract students.</Text></View>}
      {dash?.upcoming?.length>0 && <View style={styles.card}><Text style={styles.cardTitle}>Upcoming Sessions</Text>{dash.upcoming.map((b:any)=><View key={b.id} style={styles.row}><Text style={styles.body}>{new Date(b.startTime).toLocaleString()}</Text><Text style={styles.sub}>{b.status} {b.isTrial?'(trial)':''}</Text></View>)}</View>}
      {dash?.students?.length>0 && <View style={styles.card}><Text style={styles.cardTitle}>Recent Students</Text>{dash.students.slice(0,3).map((s:any,i:number)=><View key={i} style={styles.row}><Text style={styles.body}>{s.displayName||s.studentName||s.userId?.slice(0,6)}</Text></View>)}</View>}
      <Pressable style={styles.primaryBtn} onPress={()=>nav.navigate('Payouts')}><Text style={styles.primaryBtnText}>Payouts</Text></Pressable>
    </ScrollView>
  );
}
const styles = StyleSheet.create({
  container:{flex:1,backgroundColor:COLOR.background},
  content:{padding:SPACING.marginMobile,gap:SPACING.stackMd,paddingBottom:32},
  center:{flex:1,backgroundColor:COLOR.background,alignItems:'center',justifyContent:'center',padding:24},
  title:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:18},
  welcome:{backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:SPACING.stackMd,...SHADOWS.elevation1},
  welcomeTitle:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline},
  welcomeSub:{...TYPOGRAPHY.bodySm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body,marginTop:2},
  card:{backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:SPACING.stackMd,gap:6,...SHADOWS.elevation1},
  cardDark:{backgroundColor:COLOR.primary,borderRadius:RADIUS.xl,padding:SPACING.stackMd,gap:8},
  cardTitle:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:16},
  cardTitleLight:{...TYPOGRAPHY.headlineSm,color:COLOR.onPrimary,fontFamily:FONTS.headline,fontSize:16},
  body:{...TYPOGRAPHY.bodySm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body},
  bodyLight:{...TYPOGRAPHY.bodySm,color:'rgba(255,255,255,0.9)',fontFamily:FONTS.body},
  sub:{...TYPOGRAPHY.labelSm,color:COLOR.outline,fontFamily:FONTS.label},
  link:{...TYPOGRAPHY.labelMd,color:COLOR.primary,fontFamily:FONTS.label,marginTop:8},
  row:{borderTopWidth:1,borderTopColor:COLOR.outlineVariant,paddingTop:8,gap:2},
  statsRow:{flexDirection:'row',gap:8,marginTop:8},
  stat:{flex:1,backgroundColor:COLOR.background,borderRadius:12,padding:10,alignItems:'center'},
  statLabel:{...TYPOGRAPHY.labelSm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.label,fontSize:10},
  statValue:{...TYPOGRAPHY.headlineSm,color:COLOR.primary,fontFamily:FONTS.headline,fontSize:16,marginTop:4},
  track:{height:6,backgroundColor:COLOR.surfaceVariant,borderRadius:3,overflow:'hidden',marginTop:8},
  fill:{height:6,backgroundColor:COLOR.primary,borderRadius:3},
  lightBtn:{marginTop:8,backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.full,paddingVertical:10,alignItems:'center'},
  lightBtnText:{color:COLOR.primary,fontFamily:FONTS.label,fontSize:12,fontWeight:'600'},
  primaryBtn:{backgroundColor:COLOR.primary,borderRadius:RADIUS.full,paddingVertical:14,alignItems:'center'},
  primaryBtnText:{...TYPOGRAPHY.labelMd,color:COLOR.onPrimary,fontFamily:FONTS.label},
});

import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, Pressable } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';

export default function TrialCreditsScreen() {
  const nav = useNavigation<any>();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  const [tutors, setTutors] = useState<any[]>([]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const res = await (apiService as any).getTrialCreditsDashboard?.() ?? await (apiService as any).getTrialCredits?.();
        if (active) setData(res.dashboard ?? res);
        try { const r = await apiService.browseTutors({ limit: 2 }); if (active) setTutors(r.tutors.slice(0,2)); } catch {}
      } catch (e:any) { if(active) setErr(e?.response?.data?.error||e.message); }
      finally { if(active) setLoading(false); }
    })();
    return () => { active = false; };
  }, []);

  if (loading) return <View style={styles.center}><ActivityIndicator color={COLOR.primary} /></View>;
  if (err) return <View style={styles.center}><Text style={styles.body}>{err}</Text></View>;

  const credits = data?.credits ?? 0;
  const history = data?.history ?? [];
  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Trial Credits</Text>
      <View style={styles.hero}>
        <Text style={styles.star}>★</Text>
        <Text style={styles.heroNum}>{credits}</Text>
        <Text style={styles.heroLabel}>Trial credits available</Text>
        {data?.nextGrantAt && <Text style={styles.heroSub}>Next: {new Date(data.nextGrantAt).toLocaleDateString()}</Text>}
        <Pressable style={styles.findBtn} onPress={()=>nav.navigate('BrowseTutors')}><Text style={styles.findBtnText}>⌕ Find a Tutor</Text></Pressable>
      </View>
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>How Trials Work</Text>
        <View style={styles.howRow}>
          <View style={styles.howCard}><Text style={styles.howIcon}>⏱</Text><Text style={styles.howTitle}>20 Minutes</Text><Text style={styles.howSub}>Focused 1-on-1 to assess level.</Text></View>
          <View style={styles.howCard}><Text style={styles.howIcon}>🤝</Text><Text style={styles.howTitle}>Meet & Greet</Text><Text style={styles.howSub}>Casual chat to see if match.</Text></View>
        </View>
      </View>
      {tutors.length>0 && (
        <View style={styles.section}>
          <View style={styles.headRow}><Text style={styles.sectionTitle}>Recommended for Trials</Text><Pressable onPress={()=>nav.navigate('BrowseTutors')}><Text style={styles.link}>See all</Text></Pressable></View>
          {tutors.map((t:any)=><Pressable key={t.userId} style={styles.tutorRow} onPress={()=>nav.navigate('TutorProfile',{userId:t.userId})}>
            <View style={styles.avatar}><Text style={styles.avatarText}>{(t.displayName||t.userId).slice(0,1).toUpperCase()}</Text></View>
            <View style={{flex:1}}><Text style={styles.tutorName}>{t.displayName}</Text><Text style={styles.tutorSub}>{(t.languages||[]).slice(0,2).join(' • ')}</Text></View>
            <Pressable style={styles.bookBtn} onPress={()=>nav.navigate('ConfirmBooking',{userId:t.userId})}><Text style={styles.bookBtnText}>Book Trial</Text></Pressable>
          </Pressable>)}
        </View>
      )}
      <View style={styles.card}>
        <Text style={styles.cardTitle}>History</Text>
        {history.length===0 ? <Text style={styles.body}>No trial bookings yet.</Text> : history.map((b:any)=><View key={b.id} style={styles.row}><Text style={styles.body}>{new Date(b.startTime).toLocaleDateString()} · {b.status}</Text><Text style={styles.sub}>{b.isTrial?'Trial':''}</Text></View>)}
      </View>
    </ScrollView>
  );
}
const styles = StyleSheet.create({
  container:{flex:1,backgroundColor:COLOR.background},
  content:{padding:SPACING.marginMobile,gap:SPACING.stackMd,paddingBottom:32},
  center:{flex:1,backgroundColor:COLOR.background,alignItems:'center',justifyContent:'center',padding:24},
  title:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:18},
  hero:{backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:SPACING.stackMd,alignItems:'center',gap:4,...SHADOWS.elevation1},
  star:{color:COLOR.primary,fontSize:28},
  heroNum:{...TYPOGRAPHY.headlineMd,color:COLOR.primary,fontFamily:FONTS.headline,fontSize:40},
  heroLabel:{...TYPOGRAPHY.bodySm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body},
  heroSub:{...TYPOGRAPHY.labelSm,color:COLOR.outline,fontFamily:FONTS.label},
  findBtn:{marginTop:8,backgroundColor:COLOR.primary,borderRadius:RADIUS.full,paddingVertical:10,paddingHorizontal:16,alignSelf:'stretch',alignItems:'center'},
  findBtnText:{color:COLOR.onPrimary,fontFamily:FONTS.label,fontSize:13,fontWeight:'600'},
  section:{gap:8},
  sectionTitle:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:16},
  headRow:{flexDirection:'row',justifyContent:'space-between',alignItems:'center'},
  link:{color:COLOR.primary,fontFamily:FONTS.label,fontSize:12},
  howRow:{flexDirection:'row',gap:8},
  howCard:{flex:1,backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:12,gap:4,...SHADOWS.elevation1},
  howIcon:{fontSize:18},
  howTitle:{...TYPOGRAPHY.labelMd,color:COLOR.onSurface,fontFamily:FONTS.label},
  howSub:{...TYPOGRAPHY.labelSm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body,fontSize:11},
  tutorRow:{flexDirection:'row',alignItems:'center',gap:12,backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:12,...SHADOWS.elevation1},
  avatar:{width:40,height:40,borderRadius:20,backgroundColor:COLOR.surfaceVariant,alignItems:'center',justifyContent:'center'},
  avatarText:{fontWeight:'700',color:COLOR.onSurfaceVariant},
  tutorName:{...TYPOGRAPHY.labelMd,color:COLOR.onSurface,fontFamily:FONTS.label},
  tutorSub:{...TYPOGRAPHY.labelSm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body},
  bookBtn:{backgroundColor:'rgba(37,99,235,0.10)',borderRadius:8,paddingHorizontal:10,paddingVertical:6},
  bookBtnText:{color:COLOR.primary,fontFamily:FONTS.label,fontSize:11},
  card:{backgroundColor:COLOR.surfaceContainerLowest,borderRadius:RADIUS.xl,padding:SPACING.stackMd,gap:8,...SHADOWS.elevation1},
  cardTitle:{...TYPOGRAPHY.headlineSm,color:COLOR.onSurface,fontFamily:FONTS.headline,fontSize:16},
  body:{...TYPOGRAPHY.bodySm,color:COLOR.onSurfaceVariant,fontFamily:FONTS.body},
  sub:{...TYPOGRAPHY.labelSm,color:COLOR.outline,fontFamily:FONTS.label},
  row:{flexDirection:'row',justifyContent:'space-between',borderTopWidth:1,borderTopColor:COLOR.outlineVariant,paddingTop:8},
});

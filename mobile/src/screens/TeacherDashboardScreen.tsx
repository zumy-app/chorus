import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, Pressable, TextInput } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { COLOR, FONTS, TYPOGRAPHY, SPACING, RADIUS, SHADOWS } from '../theme';
import apiService from '../services/api';
import { apiErrorMessage } from '@chorus/shared';
import type { TeacherAssignment } from '@chorus/shared';

export default function TeacherDashboardScreen() {
  const nav = useNavigation<any>();
  const [dash, setDash] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');

  useEffect(()=>{ let a=true; (async()=>{ try{ const r = await (apiService as any).getTeacherDashboard(); if(a) setDash(r.dashboard ?? r);} catch(e:any){ if(a) setErr(apiErrorMessage(e));} finally{ if(a) setLoading(false);} })(); return()=>{a=false};},[]);

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
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Availability</Text>
        {dash?.upcomingAvailability?.length > 0 || dash?.availability?.length > 0 ? (dash?.upcomingAvailability ?? dash?.availability ?? []).slice(0,3).map((a:any,i:number)=><View key={i} style={styles.row}><Text style={styles.body}>{new Date(a.startTime).toLocaleString()} - {new Date(a.endTime).toLocaleTimeString()}</Text></View>) : <Text style={styles.sub}>No availability set.</Text>}
      </View>
      {dash?.upcoming?.length>0 && <View style={styles.card}><Text style={styles.cardTitle}>Upcoming Sessions</Text>{dash.upcoming.map((b:any)=><View key={b.id} style={styles.row}><Text style={styles.body}>{new Date(b.startTime).toLocaleString()}</Text><Text style={styles.sub}>{b.status} {b.isTrial?'(trial)':''}</Text></View>)}</View>}
      {dash?.students?.length>0 && <View style={styles.card}><Text style={styles.cardTitle}>Recent Students</Text>{dash.students.slice(0,3).map((s:any,i:number)=><View key={i} style={styles.row}><Text style={styles.body}>{s.displayName||s.studentName||s.userId?.slice(0,6)}</Text></View>)}</View>}
      {(!dash?.students || dash.students.length===0) && <View style={styles.card}><Text style={styles.cardTitle}>Recent Students</Text><Text style={styles.sub}>No students yet.</Text></View>}
      <TeacherAssignmentsCard />
      <Pressable style={styles.primaryBtn} onPress={()=>nav.navigate('Payouts')}><Text style={styles.primaryBtnText}>Payouts</Text></Pressable>
    </ScrollView>
  );
}

// V3 Phase 5: assignment management — create work for tutored students, review
// submissions (AI feedback arrives asynchronously via the grading queue).
function TeacherAssignmentsCard() {
  const [assignments, setAssignments] = useState<TeacherAssignment[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [formErr, setFormErr] = useState('');
  const [studentId, setStudentId] = useState('');
  const [title, setTitle] = useState('');
  const [instructions, setInstructions] = useState('');
  const [type, setType] = useState('writing');
  const [reviewFor, setReviewFor] = useState<string | null>(null);
  const [score, setScore] = useState('');
  const [feedback, setFeedback] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const list = await apiService.listTeacherAssignments();
      setAssignments(list as TeacherAssignment[]);
    } catch {
      // dashboard stays usable without the assignment roster
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const create = useCallback(async () => {
    if (!studentId.trim() || !title.trim() || creating) return;
    setCreating(true);
    setFormErr('');
    try {
      await apiService.createAssignment({
        studentId: studentId.trim(),
        type: type as any,
        title: title.trim(),
        instructions: instructions.trim(),
        content: { text: instructions.trim() || title.trim() },
        targetLanguage: 'es',
        nativeLanguage: 'en',
      });
      setStudentId('');
      setTitle('');
      setInstructions('');
      setShowForm(false);
      load();
    } catch (e: any) {
      setFormErr(apiErrorMessage(e));
    } finally {
      setCreating(false);
    }
  }, [studentId, title, instructions, type, creating, load]);

  const review = useCallback(async (id: string) => {
    if (busy) return;
    setBusy(true);
    try {
      await apiService.reviewAssignment(id, { feedback: feedback.trim() || 'Reviewed.' }, score.trim() ? Number(score) : undefined);
      setReviewFor(null);
      setScore('');
      setFeedback('');
      load();
    } catch {
      // keep the form open on failure
    } finally {
      setBusy(false);
    }
  }, [feedback, score, busy, load]);

  return (
    <View style={styles.card}>
      <View style={styles.rowHead}>
        <Text style={styles.cardTitle}>Assignments</Text>
        <Pressable style={styles.miniBtn} onPress={() => setShowForm(!showForm)}>
          <Text style={styles.miniBtnText}>{showForm ? 'Close' : '+ New'}</Text>
        </Pressable>
      </View>

      {showForm ? (
        <View style={styles.formBox}>
          <TextInput style={styles.input} value={studentId} onChangeText={setStudentId} placeholder="Student ID" placeholderTextColor={COLOR.outline} autoCapitalize="none" />
          <TextInput style={styles.input} value={title} onChangeText={setTitle} placeholder="Title (e.g. Weekend essay)" placeholderTextColor={COLOR.outline} />
          <TextInput style={styles.input} value={instructions} onChangeText={setInstructions} placeholder="Instructions" placeholderTextColor={COLOR.outline} multiline />
          <View style={styles.typeRow}>
            {['writing', 'reading', 'vocabulary_push', 'mixed'].map((t) => (
              <Pressable key={t} style={[styles.typeChip, type === t && styles.typeChipOn]} onPress={() => setType(t)}>
                <Text style={[styles.typeChipText, type === t && styles.typeChipTextOn]}>{t.replace('_', ' ')}</Text>
              </Pressable>
            ))}
          </View>
          {formErr ? <Text style={styles.errText}>{formErr}</Text> : null}
          <Pressable style={[styles.miniBtnWide, (!studentId.trim() || !title.trim() || creating) && styles.dim]} disabled={!studentId.trim() || !title.trim() || creating} onPress={create}>
            <Text style={styles.miniBtnTextLight}>{creating ? 'Creating…' : 'Create assignment'}</Text>
          </Pressable>
        </View>
      ) : null}

      {assignments.length === 0 ? (
        <Text style={styles.sub}>No assignments yet. Create work for your students.</Text>
      ) : (
        assignments.slice(0, 6).map((a) => (
          <View key={a.id} style={styles.row}>
            <View style={styles.rowHead}>
              <Text style={styles.bodyDark}>{a.title}</Text>
              <Text style={styles.sub}>{a.status}</Text>
            </View>
            <Text style={styles.sub}>{a.type.replace('_', ' ')} · {a.targetLanguage.toUpperCase()}</Text>
            {a.submission?.aiFeedback ? (
              <Text style={styles.sub} numberOfLines={2}>AI: {(a.submission.aiFeedback as any)?.feedback ?? ''}</Text>
            ) : null}
            {a.status === 'submitted' ? (
              reviewFor === a.id ? (
                <View style={styles.formBox}>
                  <TextInput style={styles.input} value={score} onChangeText={setScore} placeholder="Score (0-10)" placeholderTextColor={COLOR.outline} keyboardType="numeric" />
                  <TextInput style={styles.input} value={feedback} onChangeText={setFeedback} placeholder="Your feedback" placeholderTextColor={COLOR.outline} multiline />
                  <Pressable style={[styles.miniBtnWide, busy && styles.dim]} disabled={busy} onPress={() => review(a.id)}>
                    <Text style={styles.miniBtnTextLight}>Submit review</Text>
                  </Pressable>
                </View>
              ) : (
                <Pressable style={styles.miniBtn} onPress={() => setReviewFor(a.id)}>
                  <Text style={styles.miniBtnText}>Review</Text>
                </Pressable>
              )
            ) : null}
          </View>
        ))
      )}
    </View>
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
  bodyDark:{...TYPOGRAPHY.bodyMd,color:COLOR.onSurface,fontFamily:FONTS.body},
  bodyLight:{...TYPOGRAPHY.bodySm,color:'rgba(255,255,255,0.9)',fontFamily:FONTS.body},
  sub:{...TYPOGRAPHY.labelSm,color:COLOR.outline,fontFamily:FONTS.label},
  link:{...TYPOGRAPHY.labelMd,color:COLOR.primary,fontFamily:FONTS.label,marginTop:8},
  row:{borderTopWidth:1,borderTopColor:COLOR.outlineVariant,paddingTop:8,gap:2},
  rowHead:{flexDirection:'row',justifyContent:'space-between',alignItems:'center'},
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
  formBox:{gap:8,marginTop:6},
  input:{backgroundColor:COLOR.surfaceContainer,borderRadius:RADIUS.lg,paddingHorizontal:SPACING.stackMd,paddingVertical:10,...TYPOGRAPHY.bodyMd,color:COLOR.onSurface,fontFamily:FONTS.body},
  typeRow:{flexDirection:'row',flexWrap:'wrap',gap:6},
  typeChip:{backgroundColor:COLOR.surfaceContainer,borderRadius:999,paddingHorizontal:12,paddingVertical:6},
  typeChipOn:{backgroundColor:COLOR.primary},
  typeChipText:{...TYPOGRAPHY.labelSm,color:COLOR.onSurface,fontFamily:FONTS.label,textTransform:'capitalize'},
  typeChipTextOn:{...TYPOGRAPHY.labelSm,color:COLOR.onPrimary,fontFamily:FONTS.label,textTransform:'capitalize'},
  miniBtn:{backgroundColor:'rgba(0,74,198,0.1)',borderRadius:999,paddingHorizontal:12,paddingVertical:6},
  miniBtnWide:{backgroundColor:COLOR.primary,borderRadius:999,paddingHorizontal:16,paddingVertical:10,alignItems:'center'},
  miniBtnText:{...TYPOGRAPHY.labelMd,color:COLOR.primary,fontFamily:FONTS.label},
  miniBtnTextLight:{...TYPOGRAPHY.labelMd,color:COLOR.onPrimary,fontFamily:FONTS.label},
  errText:{...TYPOGRAPHY.bodySm,color:'#BA1A1A',fontFamily:FONTS.body},
  dim:{opacity:0.4},
});

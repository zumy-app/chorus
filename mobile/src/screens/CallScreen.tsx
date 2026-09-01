import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, TextInput } from 'react-native';
import { COLOR } from '../theme';
import apiService from '../services/api';
import webSocketService from '../services/websocket';
import type { TranscriptSegment } from '@chorus/shared';
import storage from '../utils/storage';

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60).toString().padStart(2, '0');
  const s = (sec % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}

export default function CallScreen({ route, navigation }: any) {
  const { callId, chatName, initialType } = route.params as { callId: string; chatId: string; chatName: string; initialType?: 'audio' | 'video' };
  const [status, setStatus] = useState<'active' | 'ended'>('active');
  const [muted, setMuted] = useState(false);
  const [cameraOn, setCameraOn] = useState(initialType === 'video');
  const [callType, setCallType] = useState<'audio' | 'video'>(initialType || 'audio');
  const [screenSharing, setScreenSharing] = useState(false);
  const [remoteScreenShare, setRemoteScreenShare] = useState(false);
  const [dualView, setDualView] = useState(false);
  const [immersive, setImmersive] = useState(true);
  const [speakerOn, setSpeakerOn] = useState(true);
  const [duration, setDuration] = useState(0);
  const [segments, setSegments] = useState<TranscriptSegment[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [captionInput, setCaptionInput] = useState('');
  const [sending, setSending] = useState(false);
  const [nativeLanguage, setNativeLanguage] = useState('en');
  const [bookmarked, setBookmarked] = useState<Set<number>>(new Set());

  const isVideo = callType === 'video';

  useEffect(() => {
    storage.getItem('user').then(v => {
      if (v) try { const u = JSON.parse(v); if (u.nativeLanguage) setNativeLanguage(u.nativeLanguage); } catch {}
    });
  }, []);

  useEffect(() => {
    const id = setInterval(() => setDuration(d => d + 1), 1000);
    return () => clearInterval(id);
  }, []);

  const loadCaptions = useCallback(async (offset = 0) => {
    try {
      const res = await apiService.getCaptions(callId, { limit: 50, offset });
      if (offset === 0) setSegments(res.segments);
      else setSegments(prev => [...prev, ...res.segments]);
      setHasMore(res.hasMore);
    } catch {}
  }, [callId]);

  useEffect(() => { loadCaptions(0); }, [loadCaptions]);

  useEffect(() => {
    const unsub = webSocketService.onMessage((msg: any) => {
      if (msg.type === 'live_caption') {
        const seg = msg.data?.segment as TranscriptSegment | undefined;
        if (seg) setSegments(prev => [...prev, seg]);
      }
      if (msg.type === 'call_ended') setStatus('ended');
      if (msg.type === 'webrtc_signal') {
        const t = msg.data?.type;
        if (t === 'screen-share-start') setRemoteScreenShare(true);
        if (t === 'screen-share-stop') setRemoteScreenShare(false);
        if (t === 'video-toggle' && msg.data?.callId === callId) {
          if (typeof msg.data?.enabled === 'boolean' && !msg.data.enabled) {}
        }
      }
    });
    return () => unsub();
  }, [callId]);

  const handleEnd = async () => {
    try { await apiService.endCall(callId); } catch {}
    setStatus('ended');
    setTimeout(() => navigation.goBack(), 600);
  };

  const handleSendCaption = async () => {
    const text = captionInput.trim();
    if (!text || sending) return;
    setSending(true);
    try {
      const seg = await apiService.postCaption(callId, { text, language: nativeLanguage });
      if (seg) setSegments(prev => [...prev, seg as TranscriptSegment]);
      setCaptionInput('');
    } catch {} finally { setSending(false); }
  };

  const handleBookmark = async (idx: number) => {
    if (bookmarked.has(idx)) return;
    try {
      await apiService.bookmarkCaption(callId, idx);
      setBookmarked(prev => new Set(prev).add(idx));
    } catch {}
  };

  const toggleScreenShare = () => {
    const next = !screenSharing;
    setScreenSharing(next);
    setCallType('video');
    if (!next) setRemoteScreenShare(false);
  };

  const toggleCamera = () => {
    const next = !cameraOn;
    setCameraOn(next);
    if (next) setCallType('video');
  };

  const renderCaption = ({ item, index }: { item: TranscriptSegment; index: number }) => {
    const translation = item.translations?.[nativeLanguage] || Object.values(item.translations || {})[0];
    return (
      <View style={styles.captionBubble}>
        <Text style={styles.captionOriginal}>{item.originalText}</Text>
        {translation && translation !== item.originalText ? <Text style={styles.captionTranslated}>{translation}</Text> : null}
        <View style={styles.captionFooter}>
          <Text style={styles.captionTime}>{new Date(item.startTime * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</Text>
          <TouchableOpacity onPress={() => handleBookmark(index)} style={[styles.bookmarkBtn, bookmarked.has(index) && styles.bookmarkBtnDone]}>
            <Text style={[styles.bookmarkText, bookmarked.has(index) && styles.bookmarkTextDone]}>{bookmarked.has(index) ? '✓ Saved' : '☆ Save'}</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  };

  const latestImmersive = segments.slice(-2);

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <View style={styles.avatar}><Text style={styles.avatarText}>{chatName?.charAt(0)?.toUpperCase() || '?'}</Text></View>
        <View style={styles.headerInfo}>
          <Text style={styles.headerName}>{chatName}</Text>
          <Text style={styles.headerSub}>{status === 'active' ? formatDuration(duration) : 'Ended'} · {callType === 'video' ? 'Video' : 'Audio'} call {screenSharing ? '· Sharing' : ''} {remoteScreenShare ? '· Remote sharing' : ''}</Text>
        </View>
        {isVideo && <TouchableOpacity onPress={() => setDualView(v => !v)} style={styles.minimizeBtn}><Text style={styles.minimizeText}>{dualView ? '◫' : '▣'}</Text></TouchableOpacity>}
        <TouchableOpacity onPress={() => navigation.goBack()} style={styles.minimizeBtn}><Text style={styles.minimizeText}>✕</Text></TouchableOpacity>
      </View>

      <View style={styles.hero}>
        {isVideo ? (
          <View style={dualView ? styles.videoDual : styles.videoPip}>
            <View style={dualView ? styles.videoPane : styles.videoRemote}>
              <View style={styles.videoPlaceholder}><Text style={styles.videoPlaceholderText}>{chatName?.charAt(0)?.toUpperCase() || '?'}</Text><Text style={styles.videoLabel}>{remoteScreenShare ? 'Presenting...' : 'Remote video'}</Text></View>
              <Text style={styles.paneLabel}>{chatName}</Text>
            </View>
            <View style={dualView ? styles.videoPane : styles.videoLocal}>
              {screenSharing ? (
                <View style={[styles.videoPlaceholder, { backgroundColor: '#0f2a1a' }]}><Text style={styles.videoPlaceholderText}>🖥️</Text><Text style={styles.videoLabel}>Your screen</Text></View>
              ) : cameraOn ? (
                <View style={styles.videoPlaceholder}><Text style={styles.videoPlaceholderText}>{'You'.charAt(0)}</Text><Text style={styles.videoLabel}>You</Text></View>
              ) : (
                <View style={[styles.videoPlaceholder, { backgroundColor: '#2a1a2a' }]}><Text style={styles.videoPlaceholderText}>🚫</Text><Text style={styles.videoLabel}>Cam off</Text></View>
              )}
            </View>
            {immersive && latestImmersive.length > 0 && (
              <View style={styles.immersiveOverlay} testID="immersive-captions">
                {latestImmersive.map((seg, i) => {
                  const tr = seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0];
                  return <Text key={i} style={styles.immersiveText}>{seg.originalText}{tr && tr !== seg.originalText ? ` · ${tr}` : ''}</Text>;
                })}
              </View>
            )}
          </View>
        ) : (
          <>
            <View style={styles.heroAvatar}><Text style={styles.heroAvatarText}>{chatName?.charAt(0)?.toUpperCase() || '?'}</Text></View>
            <Text style={styles.heroName}>{chatName}</Text>
            <Text style={styles.heroStatus}>{status === 'active' ? 'On call' : 'Call ended'} · {formatDuration(duration)}</Text>
            <View style={styles.liveBadge}><View style={styles.liveDot} /><Text style={styles.liveText}>Live</Text></View>
            {immersive && latestImmersive.length > 0 && (
              <View style={[styles.immersiveOverlay, { position: 'relative', marginTop: 12 }]} testID="immersive-captions">
                {latestImmersive.map((seg, i) => {
                  const tr = seg.translations?.[nativeLanguage] || Object.values(seg.translations || {})[0];
                  return <Text key={i} style={styles.immersiveText}>{seg.originalText}{tr && tr !== seg.originalText ? ` · ${tr}` : ''}</Text>;
                })}
              </View>
            )}
          </>
        )}
      </View>

      <View style={styles.transcriptWrap}>
        <View style={styles.transcriptHeader}>
          <Text style={styles.transcriptTitle}>Live captions</Text>
          <View style={{ flexDirection: 'row', gap: 12 }}>
            <TouchableOpacity onPress={() => setImmersive(v => !v)}><Text style={[styles.transcriptHint, immersive && { color: '#a78bfa' }]}>{immersive ? 'Immersive on' : 'Immersive off'}</Text></TouchableOpacity>
            <Text style={styles.transcriptHint}>Scrollable · tap Save</Text>
          </View>
        </View>
        {segments.length === 0 ? (
          <View style={styles.empty}><Text style={styles.emptyIcon}>💬</Text><Text style={styles.emptyText}>Captions will appear here as you speak.</Text></View>
        ) : (
          <FlatList data={segments} renderItem={renderCaption} keyExtractor={(_, i) => String(i)} style={styles.list} contentContainerStyle={styles.listContent} />
        )}
        {hasMore ? <TouchableOpacity onPress={() => loadCaptions(segments.length)} style={styles.loadMore}><Text style={styles.loadMoreText}>Load older captions</Text></TouchableOpacity> : null}
        <View style={styles.inputRow}>
          <TextInput value={captionInput} onChangeText={setCaptionInput} placeholder="Type a caption..." placeholderTextColor={COLOR.onSurfaceVariant} style={styles.input} onSubmitEditing={handleSendCaption} returnKeyType="send" />
          <TouchableOpacity onPress={handleSendCaption} disabled={!captionInput.trim() || sending} style={[styles.sendBtn, (!captionInput.trim() || sending) && styles.sendBtnDisabled]}><Text style={styles.sendBtnText}>➤</Text></TouchableOpacity>
        </View>
      </View>

      <View style={styles.controls}>
        <TouchableOpacity onPress={() => setMuted(v => !v)} style={[styles.ctrlBtn, muted && styles.ctrlBtnActive]}><Text style={[styles.ctrlIcon, muted && styles.ctrlIconActive]}>{muted ? '🔇' : '🎤'}</Text></TouchableOpacity>
        <TouchableOpacity onPress={toggleCamera} style={[styles.ctrlBtn, !cameraOn && styles.ctrlBtnMuted]}><Text style={styles.ctrlIcon}>{cameraOn ? '📹' : '🚫'}</Text></TouchableOpacity>
        <TouchableOpacity onPress={toggleScreenShare} style={[styles.ctrlBtn, screenSharing && styles.ctrlBtnActive]}><Text style={styles.ctrlIcon}>🖥️</Text></TouchableOpacity>
        <TouchableOpacity onPress={() => setSpeakerOn(v => !v)} style={[styles.ctrlBtn, !speakerOn && styles.ctrlBtnMuted]}><Text style={styles.ctrlIcon}>{speakerOn ? '🔊' : '🔈'}</Text></TouchableOpacity>
        <TouchableOpacity onPress={handleEnd} style={styles.endBtn}><Text style={styles.endIcon}>✕</Text><Text style={styles.endText}>End</Text></TouchableOpacity>
      </View>

      {status === 'ended' ? (
        <View style={styles.endedOverlay}>
          <View style={styles.endedCard}>
            <Text style={styles.endedTitle}>Call ended</Text><Text style={styles.endedSub}>Duration {formatDuration(duration)}</Text>
            <TouchableOpacity onPress={() => navigation.goBack()} style={styles.endedClose}><Text style={styles.endedCloseText}>Close</Text></TouchableOpacity>
          </View>
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0B1C30' },
  header: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 16, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.08)' },
  avatar: { width: 36, height: 36, borderRadius: 18, backgroundColor: 'rgba(255,255,255,0.12)', alignItems: 'center', justifyContent: 'center' },
  avatarText: { color: '#fff', fontWeight: '700' },
  headerInfo: { flex: 1, marginLeft: 12 },
  headerName: { color: '#fff', fontWeight: '600', fontSize: 15 },
  headerSub: { color: 'rgba(255,255,255,0.6)', fontSize: 12, marginTop: 2 },
  minimizeBtn: { width: 36, height: 36, borderRadius: 18, backgroundColor: 'rgba(255,255,255,0.1)', alignItems: 'center', justifyContent: 'center', marginLeft: 6 },
  minimizeText: { color: '#fff', fontSize: 14 },
  hero: { alignItems: 'center', paddingVertical: 16, backgroundColor: '#132a4a' },
  heroAvatar: { width: 84, height: 84, borderRadius: 42, backgroundColor: 'rgba(255,255,255,0.1)', borderWidth: 1, borderColor: 'rgba(255,255,255,0.15)', alignItems: 'center', justifyContent: 'center' },
  heroAvatarText: { color: '#fff', fontSize: 32, fontWeight: '700' },
  heroName: { color: '#fff', fontSize: 18, fontWeight: '600', marginTop: 12 },
  heroStatus: { color: 'rgba(255,255,255,0.6)', fontSize: 13, marginTop: 4 },
  liveBadge: { flexDirection: 'row', alignItems: 'center', marginTop: 10, backgroundColor: 'rgba(16,185,129,0.15)', paddingHorizontal: 10, paddingVertical: 4, borderRadius: 9999, borderWidth: 1, borderColor: 'rgba(16,185,129,0.3)' },
  liveDot: { width: 6, height: 6, borderRadius: 3, backgroundColor: '#10b981', marginRight: 6 },
  liveText: { color: '#6ee7b7', fontSize: 11, fontWeight: '700', letterSpacing: 0.5 },
  videoPip: { width: '100%', height: 220, backgroundColor: '#000', position: 'relative', overflow: 'hidden' },
  videoDual: { width: '100%', height: 220, flexDirection: 'row', gap: 4, padding: 4, backgroundColor: '#000' },
  videoPane: { flex: 1, backgroundColor: '#0B1C30', borderRadius: 12, overflow: 'hidden', position: 'relative' },
  videoRemote: { ...StyleSheet.absoluteFillObject },
  videoLocal: { position: 'absolute', bottom: 8, right: 8, width: 90, height: 120, borderRadius: 12, overflow: 'hidden', borderWidth: 2, borderColor: 'rgba(255,255,255,0.2)' },
  videoPlaceholder: { flex: 1, backgroundColor: '#1a3354', alignItems: 'center', justifyContent: 'center', gap: 4 },
  videoPlaceholderText: { color: '#fff', fontSize: 22, fontWeight: '700' },
  videoLabel: { color: 'rgba(255,255,255,0.6)', fontSize: 11 },
  paneLabel: { position: 'absolute', bottom: 4, left: 6, backgroundColor: 'rgba(0,0,0,0.6)', color: '#fff', fontSize: 10, paddingHorizontal: 6, paddingVertical: 2, borderRadius: 9999 },
  immersiveOverlay: { position: 'absolute', bottom: 8, left: 8, right: 8, backgroundColor: 'rgba(0,0,0,0.65)', borderRadius: 8, padding: 8, borderWidth: 1, borderColor: 'rgba(255,255,255,0.1)' },
  immersiveText: { color: '#fff', fontSize: 12, lineHeight: 16, textAlign: 'center' },
  transcriptWrap: { flex: 1, backgroundColor: '#0F2440', borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.08)' },
  transcriptHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingHorizontal: 16, paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.06)' },
  transcriptTitle: { color: '#fff', fontWeight: '600', fontSize: 13 },
  transcriptHint: { color: 'rgba(255,255,255,0.4)', fontSize: 11 },
  list: { flex: 1 },
  listContent: { padding: 16, gap: 10 },
  captionBubble: { backgroundColor: 'rgba(255,255,255,0.06)', borderWidth: 1, borderColor: 'rgba(255,255,255,0.08)', borderRadius: 12, padding: 12 },
  captionOriginal: { color: '#fff', fontSize: 14, lineHeight: 20 },
  captionTranslated: { color: '#C4B5FD', fontSize: 13, fontStyle: 'italic', marginTop: 6, borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.08)', paddingTop: 6 },
  captionFooter: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 8 },
  captionTime: { color: 'rgba(255,255,255,0.4)', fontSize: 11 },
  bookmarkBtn: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 9999, backgroundColor: 'rgba(255,255,255,0.08)', borderWidth: 1, borderColor: 'rgba(255,255,255,0.1)' },
  bookmarkBtnDone: { backgroundColor: 'rgba(16,185,129,0.15)', borderColor: 'rgba(16,185,129,0.3)' },
  bookmarkText: { color: 'rgba(255,255,255,0.8)', fontSize: 11, fontWeight: '600' },
  bookmarkTextDone: { color: '#6ee7b7' },
  empty: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32 },
  emptyIcon: { fontSize: 28, marginBottom: 8 },
  emptyText: { color: 'rgba(255,255,255,0.4)', fontSize: 13, textAlign: 'center' },
  loadMore: { alignItems: 'center', paddingVertical: 8 },
  loadMoreText: { color: 'rgba(255,255,255,0.5)', fontSize: 12 },
  inputRow: { flexDirection: 'row', padding: 12, borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.08)', gap: 8 },
  input: { flex: 1, backgroundColor: 'rgba(255,255,255,0.08)', borderWidth: 1, borderColor: 'rgba(255,255,255,0.12)', borderRadius: 9999, paddingHorizontal: 16, paddingVertical: 10, color: '#fff', fontSize: 14 },
  sendBtn: { width: 40, height: 40, borderRadius: 20, backgroundColor: COLOR.primary, alignItems: 'center', justifyContent: 'center' },
  sendBtnDisabled: { opacity: 0.4 },
  sendBtnText: { color: '#fff', fontSize: 16 },
  controls: { flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 8, paddingVertical: 14, borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.08)', backgroundColor: '#081428', flexWrap: 'wrap' },
  ctrlBtn: { width: 44, height: 44, borderRadius: 22, backgroundColor: 'rgba(255,255,255,0.08)', alignItems: 'center', justifyContent: 'center' },
  ctrlBtnActive: { backgroundColor: '#10b981' },
  ctrlBtnMuted: { opacity: 0.5 },
  ctrlIcon: { fontSize: 16 },
  ctrlIconActive: { color: '#fff' },
  endBtn: { flexDirection: 'row', alignItems: 'center', gap: 6, backgroundColor: '#dc2626', paddingHorizontal: 18, height: 44, borderRadius: 22 },
  endIcon: { color: '#fff', fontSize: 14, fontWeight: '700' },
  endText: { color: '#fff', fontWeight: '700' },
  endedOverlay: { ...StyleSheet.absoluteFillObject, backgroundColor: 'rgba(11,28,48,0.8)', alignItems: 'center', justifyContent: 'center', padding: 24 },
  endedCard: { backgroundColor: '#fff', borderRadius: 16, padding: 24, alignItems: 'center', width: '100%', maxWidth: 320 },
  endedTitle: { fontSize: 18, fontWeight: '700', color: '#0B1C30' },
  endedSub: { fontSize: 13, color: '#6b7280', marginTop: 4 },
  endedClose: { marginTop: 16, backgroundColor: COLOR.primary, paddingHorizontal: 24, paddingVertical: 10, borderRadius: 9999 },
  endedCloseText: { color: '#fff', fontWeight: '600' },
});

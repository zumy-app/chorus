import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
  Modal,
  Linking,
  Alert,
  ScrollView,
} from 'react-native';
import storage from '../utils/storage';
import apiService from '../services/api';
import featureFlags from '../utils/featureFlags';
import { useStrings, t, useAppLocale } from '../i18n';
import webSocketService from '../services/websocket';
import { Message, WebSocketMessage, User, apiErrorMessage } from '@chorus/shared';
import { COLOR, FONTS } from '../theme';
import RealTalkNudge from '../components/RealTalkNudge';

export default function ChatScreen({ route, navigation }: any) {
  const { chatId, chatName } = route.params;
  const s = useStrings();
  const locale = useAppLocale();
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputText, setInputText] = useState('');
  const [loading, setLoading] = useState(true);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [typing, setTyping] = useState(false);
  const [translatingId, setTranslatingId] = useState<string | null>(null);
  const [translateAsType, setTranslateAsType] = useState(false);
  const [deepDiveVisible, setDeepDiveVisible] = useState(false);
  const [sparkyInput, setSparkyInput] = useState('');
  const [sparkyMessages, setSparkyMessages] = useState<{ role: 'user' | 'assistant'; content: string; jobId?: string }[]>([]);
  const [sparkyLoading, setSparkyLoading] = useState(false);
  const [sparkyError, setSparkyError] = useState('');
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [pinned, setPinned] = useState<any[]>([]);
  const [pinnedOpen, setPinnedOpen] = useState(false);
  const [forwardMsg, setForwardMsg] = useState<Message | null>(null);
  const [forwardChats, setForwardChats] = useState<any[]>([]);
  const [actionMsg, setActionMsg] = useState<Message | null>(null);
  const [uploading, setUploading] = useState(false);
  const [otherUserId, setOtherUserId] = useState<string | null>(null);
  const [otherName, setOtherName] = useState<string>('');
  const [reportVisible, setReportVisible] = useState(false);
  const [reportTarget, setReportTarget] = useState<{ type: 'user' | 'message'; userId?: string; messageId?: string; chatId?: string } | null>(null);
  const [reportReason, setReportReason] = useState('spam');
  const [isBlocked, setIsBlocked] = useState(false);
  const [callsEnabled, setCallsEnabled] = useState(featureFlags.isEnabled('video_calls'));
  // Release-gated composer attachments (fail-closed: hidden until flags load).
  const [mediaEnabled, setMediaEnabled] = useState(featureFlags.isEnabled('media_sharing'));
  const [docsEnabled, setDocsEnabled] = useState(featureFlags.isEnabled('document_sharing'));
  const [locationEnabled, setLocationEnabled] = useState(featureFlags.isEnabled('location_sharing'));
  const [voiceEnabled, setVoiceEnabled] = useState(featureFlags.isEnabled('voice_message'));
  const [realTalkEnabled, setRealTalkEnabled] = useState(featureFlags.isEnabled('real_talk'));
  const [translateTypeEnabled, setTranslateTypeEnabled] = useState(featureFlags.isEnabled('translate_as_you_type'));
  // Async grammar analysis jobs keyed by messageId. The backend fans out one
  // job per learner per message; results arrive as "grammar_analysis" WS
  // events (or via GET /grammar/analyze/:jobId on resync).
  const [grammarJobs, setGrammarJobs] = useState<Record<string, any>>({});
  const [grammarModal, setGrammarModal] = useState<any | null>(null);
  const [grammarLoadingId, setGrammarLoadingId] = useState<string | null>(null);
  const flatListRef = useRef<FlatList>(null);
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latestMessageId = useRef<string | null>(null);
  // Pending async Sparky job (Phase 1). The WS "sparky_result" branch resolves
  // it by id; a ref (not state) keeps the WS subscription stable. The id is
  // also persisted so killing the app mid-answer doesn't orphan the job —
  // reopening the sheet resyncs via GET /sparky/jobs/:jobId (same pattern as
  // grammar jobs on web).
  const sparkyJobRef = useRef<string | null>(null);
  const SPARKY_PENDING_KEY = 'sparky_pending_job';

  // Reopen recovery: if a previous session left a pending job for this chat,
  // adopt it and resolve its current state immediately.
  const openSparkySheet = useCallback(async () => {
    setDeepDiveVisible(true);
    try {
      const raw = await storage.getItem(SPARKY_PENDING_KEY);
      if (!raw) return;
      const saved = JSON.parse(raw);
      if (!saved?.jobId || saved?.chatId !== chatId || sparkyJobRef.current) return;
      sparkyJobRef.current = saved.jobId;
      setSparkyLoading(true);
      try {
        const job: any = await (apiService as any).sparkyJob(saved.jobId);
        if (job?.status === 'done') {
          const content = job.content || 'Done';
          setSparkyMessages(prev => [...prev, { role: 'assistant', content, jobId: saved.jobId }]);
          setSparkyError('');
          setSparkyLoading(false);
          sparkyJobRef.current = null;
          await storage.removeItem(SPARKY_PENDING_KEY).catch(() => {});
        } else           if (job?.status === 'failed') {
          const msg = job.error || t('chat.sparkyFailB');
          setSparkyError(msg);
          setSparkyMessages(prev => [...prev, { role: 'assistant', content: msg }]);
          setSparkyLoading(false);
          sparkyJobRef.current = null;
          await storage.removeItem(SPARKY_PENDING_KEY).catch(() => {});
        }
        // processing/streaming: stay loading, the WS event will deliver.
      } catch {
        // Resync failed (offline?) — stay loading, the WS event may deliver.
      }
    } catch {
      // Corrupted storage — ignore.
    }
  }, [chatId]);

  const nativeLanguage = currentUser?.nativeLanguage || 'en';
  const targetLang = currentUser?.targetLanguages?.[0]?.toUpperCase();

  const handleWebSocket = useCallback((message: WebSocketMessage) => {
    const payload: any = message.data || {};
    if (message.type === 'new_message' && payload.chatId === chatId) {
      setMessages((prev) => [
        payload,
        ...prev.filter((m: any) => m.id !== payload.id),
      ]);
      if (payload.senderId !== currentUser?.id) {
        (webSocketService as any).send?.({ type: 'message_ack', data: { chatId, messageId: payload.id, status: 'received' } });
        setTimeout(() => (webSocketService as any).send?.({ type: 'message_ack', data: { chatId, messageId: payload.id, status: 'read' } }), 300);
        apiService.markAsRead(chatId, payload.id).catch(() => {});
      }
    } else if (message.type === 'message_updated' && payload.chatId === chatId) {
      setMessages((prev: any) =>
        prev.map((m: any) => (m.id === payload.id ? payload : m))
      );
    } else if (message.type === 'message_deleted' && payload.chatId === chatId) {
      setMessages((prev: any) => prev.filter((m: any) => m.id !== payload.messageId));
    } else if ((message.type === 'message_pinned' || message.type === 'message_unpinned') && payload.chatId === chatId) {
      apiService.getPinnedMessages(chatId).then(p => setPinned(p ?? [])).catch(()=>{});
    } else if ((message.type === 'message_delivered' || message.type === 'message_read') && payload.chatId === chatId) {
      const status = message.type === 'message_read' ? 'read' : 'delivered';
      const uid = payload.userId || '';
      setMessages((prev: any) => prev.map((m: any) => {
        if (m.id !== payload.messageId) return m;
        const receipts = m.receipts ? [...m.receipts] : [];
        const idx = receipts.findIndex((r: any) => r.userId === uid);
        if (idx >= 0) receipts[idx] = { ...receipts[idx], status };
        else receipts.push({ messageId: payload.messageId, chatId, userId: uid, status });
        return { ...m, receipts };
      }));
    } else if (message.type === 'grammar_analysis') {
      // Per-user fanout: payload carries {jobId, messageId, chatId, status,
      // analysis?, providerUsed?, error?}. Ignore events for other chats.
      const key = payload.messageId || payload.jobId;
      if (key && (!payload.chatId || payload.chatId === chatId)) {
        setGrammarJobs((prev: any) => ({ ...prev, [key]: payload }));
      }
    } else if (message.type === 'sparky_result') {
      // Async Sparky answer: {jobId, chatId, status, content?, providerUsed?,
      // error?}. Phase 2 streams tokens as status "streaming" deltas, then a
      // final "done" with the full cleaned content. Only the matching pending
      // question resolves.
      if (payload.jobId && payload.jobId === sparkyJobRef.current) {
        if (payload.status === 'streaming') {
          const delta = payload.content || '';
          if (!delta) return;
          setSparkyMessages(prev => {
            const last = prev[prev.length - 1];
            if (last && last.role === 'assistant' && (last as any).jobId === payload.jobId) {
              const next = [...prev];
              next[next.length - 1] = { ...last, content: last.content + delta };
              return next;
            }
            return [...prev, { role: 'assistant', content: delta, jobId: payload.jobId }];
          });
        } else if (payload.status === 'done') {
          const content = payload.content || 'Done';
          setSparkyMessages(prev => {
            const last = prev[prev.length - 1];
            if (last && last.role === 'assistant' && (last as any).jobId === payload.jobId) {
              // Replace the streamed partial with the full cleaned answer.
              const next = [...prev];
              next[next.length - 1] = { ...last, content };
              return next;
            }
            return [...prev, { role: 'assistant', content, jobId: payload.jobId }];
          });
          setSparkyError('');
          setSparkyLoading(false);
          sparkyJobRef.current = null;
          storage.removeItem(SPARKY_PENDING_KEY).catch(() => {});
        } else if (payload.status === 'failed') {
          const msg = payload.error || t('chat.sparkyFailB');
          setSparkyError(msg);
          setSparkyMessages(prev => [...prev, { role: 'assistant', content: msg }]);
          setSparkyLoading(false);
          sparkyJobRef.current = null;
          storage.removeItem(SPARKY_PENDING_KEY).catch(() => {});
        }
        // "processing" keeps the typing indicator until tokens arrive.
      }
    } else if (message.type === 'user_typing' && payload.chatId === chatId) {
      setTyping(payload.isTyping === true);
    } else if (message.type === 'call_incoming' && payload.chatId === chatId) {
      const callId = (payload as Record<string, string>).callId || (payload as Record<string, string>).call_id;
      const t = (payload as Record<string, string>).type || 'audio';
      if (callId) navigation.navigate('Call', { callId, chatId, chatName, initialType: t });
    }
  }, [chatId, chatName, navigation, currentUser?.id]);

  const loadCurrentUser = useCallback(async () => {
    const userStr = await storage.getItem('user');
    if (userStr) {
      try {
        setCurrentUser(JSON.parse(userStr));
      } catch {
        // Corrupted storage — ignore.
      }
    }
  }, []);

  const loadMessages = useCallback(async () => {
    try {
      const data = await apiService.getMessages(chatId);
      setMessages(data);
      if (data.length > 0) {
        latestMessageId.current = data[0].id;
        apiService.markAsRead(chatId, data[0].id).catch(()=>{});
        for (const m of data) {
          if (m.senderId !== currentUser?.id) {
            (webSocketService as any).send?.({ type: 'message_ack', data: { chatId, messageId: m.id, status: 'received' } });
          }
        }
        (webSocketService as any).send?.({ type: 'message_ack', data: { chatId, messageId: data[0].id, status: 'read' } });
      }
    } catch {
    } finally {
      setLoading(false);
    }
  }, [chatId, currentUser?.id]);

  const startCall = async (type: 'audio' | 'video' = 'audio') => {
    try {
      const res = await apiService.initiateCall(chatId, type);
      navigation.navigate('Call', { callId: res.session.id, chatId, chatName, initialType: type });
    } catch {}
  };

  const openReport = (target: { type: 'user' | 'message'; userId?: string; messageId?: string; chatId?: string }) => {
    setReportTarget(target);
    setReportReason('spam');
    setReportVisible(true);
  };
  const submitReport = async () => {
    if (!reportTarget) return;
    try {
      await (apiService as any).reportUser({ type: reportTarget.type, reportedUserId: reportTarget.userId, messageId: reportTarget.messageId, chatId: reportTarget.chatId || chatId, reason: reportReason });
      Alert.alert(t('chat.reportedT'), t('chat.reportedB'));
    } catch { Alert.alert(t('common.error'), t('chat.reportFailB')); }
    setReportVisible(false);
  };
  const toggleBlock = async () => {
    if (!otherUserId) { Alert.alert(t('chat.blockUnavailableT'), t('chat.blockUnavailableB')); return; }
    try {
      if (isBlocked) { await (apiService as any).unblockUser(otherUserId); setIsBlocked(false); Alert.alert(t('chat.unblockedT'), t('chat.unblockedB', { name: otherName || t('chat.userFallback') })); }
      else {
        Alert.alert(t('chat.blockUserT'), t('chat.blockUserB', { name: otherName || t('chat.thisUserFallback') }), [
          { text: t('common.cancel'), style: 'cancel' },
          { text: t('chat.block'), style: 'destructive', onPress: async () => { await (apiService as any).blockUser(otherUserId); setIsBlocked(true); } },
        ]);
      }
    } catch { Alert.alert(t('common.error'), t('chat.blockFailB')); }
  };
  useEffect(() => {
    let mounted = true;
    featureFlags.init().then(() => {
      if (!mounted) return;
      setCallsEnabled(featureFlags.isEnabled('video_calls'));
      setMediaEnabled(featureFlags.isEnabled('media_sharing'));
      setDocsEnabled(featureFlags.isEnabled('document_sharing'));
      setLocationEnabled(featureFlags.isEnabled('location_sharing'));
      setVoiceEnabled(featureFlags.isEnabled('voice_message'));
      setRealTalkEnabled(featureFlags.isEnabled('real_talk'));
      setTranslateTypeEnabled(featureFlags.isEnabled('translate_as_you_type'));
    });
    apiService.getChat(chatId).then(c => {
      if (!mounted) return;
      const other = (c.participants || []).find((p:any)=>p.user?.id !== currentUser?.id)?.user;
      if (other) { setOtherUserId(other.id); setOtherName(other.displayName || ''); if (other.id) (apiService as any).getBlockStatus(other.id).then((s:any)=>setIsBlocked(s.blocked)).catch(()=>{}) }
    }).catch(()=>{});
    return () => { mounted = false; };
  }, [chatId, currentUser?.id]);
  useEffect(() => {
    navigation.setOptions({
      title: chatName,
      headerTintColor: COLOR.primary,
      headerTitleStyle: { fontSize: 18, fontWeight: '600', color: COLOR.primary },
      headerRight: () => (
        <View style={{ flexDirection: 'row', gap: 6, marginRight: 4, alignItems: 'center' }}>
          {callsEnabled && (
            <>
              <TouchableOpacity onPress={() => startCall('audio')} style={{ width: 36, height: 36, borderRadius: 18, backgroundColor: COLOR.primaryContainer, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#fff', fontSize: 16 }}>📞</Text>
              </TouchableOpacity>
              <TouchableOpacity onPress={() => startCall('video')} style={{ width: 36, height: 36, borderRadius: 18, backgroundColor: COLOR.primary, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#fff', fontSize: 14 }}>📹</Text>
              </TouchableOpacity>
            </>
          )}
          <TouchableOpacity
            onPress={() => {
              Alert.alert(t('chat.chatActions'), `${otherName || t('chat.userFallback')}`, [
                { text: isBlocked ? t('chat.unblock') : t('chat.block'), style: isBlocked ? 'default' : 'destructive', onPress: toggleBlock },
                { text: t('chat.reportUser'), onPress: () => otherUserId && openReport({ type: 'user', userId: otherUserId, chatId }) },
                { text: t('common.cancel'), style: 'cancel' },
              ]);
            }}
            style={{ width: 36, height: 36, borderRadius: 18, backgroundColor: COLOR.surfaceContainerHigh, alignItems: 'center', justifyContent: 'center' }}
            testID="chat-more-actions"
          >
            <Text style={{ color: COLOR.onSurface, fontSize: 18 }}>⋮</Text>
          </TouchableOpacity>
        </View>
      ),
    });
  }, [navigation, chatName, chatId, otherName, otherUserId, isBlocked, callsEnabled]);

  useEffect(() => {
    loadCurrentUser();
    loadMessages();
    apiService.getPinnedMessages(chatId).then(p => setPinned(p ?? [])).catch(()=>{});
    webSocketService.connect();

    const unsubscribe = webSocketService.onMessage(handleWebSocket);

    return () => {
      unsubscribe();
    };
  }, [chatId, handleWebSocket, loadMessages, loadCurrentUser]);

  useEffect(() => {
    storage.getItem('realTalkDraft').then((d) => {
      if (d) {
        setInputText(d);
        storage.removeItem('realTalkDraft');
      }
    });
  }, [chatId]);

  const handleSend = async () => {
    if (!inputText.trim()) return;

    const messageText = inputText.trim();
    const replyId = replyTo?.id
    setInputText('');
    setReplyTo(null);

    try {
      const newMessage = await apiService.sendMessage(chatId, messageText, replyId);
      setMessages((prev) => [newMessage, ...prev.filter((m) => m.id !== newMessage.id)]);
      latestMessageId.current = newMessage.id;
      apiService.markAsRead(chatId, newMessage.id);
    } catch {
      setInputText(messageText);
    }
  };

  const handleShareLocation = async () => {
    const send = async (lat: number, lng: number, label?: string) => {
      try {
        setUploading(true)
        const msg = await (apiService as any).sendLocation(chatId, { latitude: lat, longitude: lng, label, replyToId: replyTo?.id })
        setMessages((prev) => [msg, ...prev.filter((m:any)=>m.id!==msg.id)])
        setReplyTo(null)
      } catch (e:any) { Alert.alert(t('chat.locFailT'), e?.response?.data?.error || String(e)) }
      finally { setUploading(false) }
    }
    try {
      let Location: any = null
      try { Location = require('expo-location') } catch {}
      if (Location?.requestForegroundPermissionsAsync) {
        const { status } = await Location.requestForegroundPermissionsAsync()
        if (status !== 'granted') { Alert.alert(t('chat.permT'), t('chat.permB')); return }
        const pos = await Location.getCurrentPositionAsync({})
        await send(pos.coords.latitude, pos.coords.longitude)
        return
      }
    } catch {}
    if (typeof (globalThis as any).navigator !== 'undefined' && ((globalThis as any).navigator as any).geolocation) {
      ((globalThis as any).navigator as any).geolocation.getCurrentPosition(
        (pos: any) => send(pos.coords.latitude, pos.coords.longitude),
        () => Alert.alert(t('chat.locUnavailableT'), t('chat.locUnavailableB'))
      )
    } else {
      Alert.alert(t('chat.locNotAvailT'), t('chat.locNotAvailB'))
    }
  }

  const handleAttach = async () => {
    try {
      let DocumentPicker: any = null
      try { DocumentPicker = require('expo-document-picker') } catch {}
      if (DocumentPicker?.getDocumentAsync) {
        const res = await DocumentPicker.getDocumentAsync({ type: ['application/pdf','application/msword','application/vnd.openxmlformats-officedocument.*','text/plain'], copyToCacheDirectory: true })
        if (res.canceled || !res.assets?.[0]) return
        const asset = res.assets[0]
        if (asset.size && asset.size > 50*1024*1024) { Alert.alert(t('chat.fileTooLargeT'), t('chat.fileTooLargeB')); return }
        setUploading(true)
        try {
          const blob = await (await fetch(asset.uri)).blob()
          const msg = await (apiService as any).sendAttachment(chatId, blob, asset.name || 'document')
          setMessages((prev) => [msg, ...prev.filter((m:any)=>m.id!==msg.id)])
        } finally { setUploading(false) }
        return
      }
      Alert.alert(t('chat.docPickerT'), t('chat.docPickerB'))
    } catch (e) { Alert.alert(t('chat.uploadFailT'), String(e)) }
  }

  const handleTyping = () => {
    webSocketService.sendTyping(chatId, true);

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    typingTimeoutRef.current = setTimeout(() => {
      webSocketService.sendTyping(chatId, false);
    }, 1500);
  };

  const handleTranslate = async (messageId: string) => {
    if (translatingId) return;
    setTranslatingId(messageId);
    try {
      await apiService.translateMessage(chatId, messageId, nativeLanguage);
    } catch {
      // Translation request failed — non-fatal.
    } finally {
      setTranslatingId(null);
    }
  };

  // Manual grammar analysis request (mirrors web MessageBubble): submits an
  // async job; the result arrives via the "grammar_analysis" WS event above
  // (or GET /grammar/analyze/:jobId on resync). Cache hot-path returns done.
  const handleGrammarAnalyze = async (item: Message) => {
    if (grammarLoadingId) return;
    setGrammarLoadingId(item.id);
    try {
      const sourceLang = (item as any).originalLanguage || 'en';
      const res: any = await (apiService as any).grammarAnalyzeAI({
        text: item.text,
        language: sourceLang,
        nativeLanguage,
        messageId: item.id,
        chatId,
      });
      if (res?.status === 'done' && res?.analysis) {
        setGrammarJobs((prev: any) => ({
          ...prev,
          [item.id]: { jobId: res.jobId || '', messageId: item.id, chatId, status: 'done', analysis: res.analysis, providerUsed: res.providerUsed || 'cache' },
        }));
      } else if (res?.jobId) {
        setGrammarJobs((prev: any) => ({
          ...prev,
          [item.id]: { jobId: res.jobId, messageId: item.id, chatId, status: res.status || 'queued' },
        }));
      }
    } catch {
      // Grammar request failed — non-fatal; badge simply stays requestable.
    } finally {
      setGrammarLoadingId(null);
    }
  };

  const renderMessage = ({ item }: { item: Message }) => {
    const isOwn = item.senderId === currentUser?.id;
    const isPinned = pinned.some((p:any)=>p.message.id===item.id)
    const replySource = (item as any).replyToId ? messages.find(m=>m.id===(item as any).replyToId) : null
    const nativeTranslation =
      item.translations?.[nativeLanguage] &&
      item.translations[nativeLanguage] !== item.text
        ? item.translations[nativeLanguage]
        : null;

    return (
      <TouchableOpacity activeOpacity={0.8} onLongPress={()=>setActionMsg(item)} delayLongPress={400}>
      <View style={[styles.messageContainer, isOwn ? styles.ownMessage : styles.otherMessage]}>
        {!isOwn && (
          <Text style={styles.senderName}>{item.sender?.displayName || 'Unknown'}</Text>
        )}
        <View style={[styles.messageBubble, isOwn ? styles.ownBubble : styles.otherBubble]}>
          {(item as any).forwarded && <Text style={[styles.forwardLabel, isOwn ? styles.ownLabel : styles.otherLabel]}>{s.chat.forwarded}</Text>}
          {isPinned && <Text style={[styles.forwardLabel, {color: '#b45309'}]}>{s.chat.pinnedLabel}</Text>}
          {replySource && (
            <View style={[styles.replyQuote, isOwn ? styles.replyQuoteOwn : styles.replyQuoteOther]}>
              <Text style={styles.replyQuoteName} numberOfLines={1}>{replySource.sender?.displayName || 'Unknown'}</Text>
              <Text style={styles.replyQuoteText} numberOfLines={1}>{replySource.text}</Text>
            </View>
          )}
          {item.media && item.media.length > 0 && item.media.map((att:any)=> att.type==='location' ? (
            <TouchableOpacity key={att.id} onPress={()=> att.url && Linking.openURL(att.url)} style={[styles.docCard, isOwn ? styles.docCardOwn : styles.docCardOther, {flexDirection:'column', alignItems:'stretch'}]}>
              <View style={{flexDirection:'row', alignItems:'center', gap:6}}>
                <Text style={styles.docIcon}>📍</Text>
                <View style={{flex:1}}>
                  <Text style={[styles.docName, isOwn?{color:'#fff'}:{}]}>{att.locationName || s.chat.sharedLocation}</Text>
                  <Text style={[styles.docMeta, isOwn?{color:'rgba(255,255,255,0.7)'}:{}]}>{att.latitude?.toFixed(5)}, {att.longitude?.toFixed(5)}</Text>
                </View>
                <Text style={styles.docDownload}>↗</Text>
              </View>
              <Text style={[styles.docMeta, {color: COLOR.primary, marginTop:4}]}>{s.chat.openInMaps}</Text>
            </TouchableOpacity>
          ) : (
            <TouchableOpacity key={att.id} onPress={()=> att.url && Linking.openURL(att.url)} style={[styles.docCard, isOwn ? styles.docCardOwn : styles.docCardOther]}>
              <Text style={styles.docIcon}>{att.type==='document'?'📄': att.type==='image'?'🖼️': att.type==='video'?'🎬': att.type==='audio'?'🎵':'📎'}</Text>
              <View style={{flex:1}}>
                <Text style={[styles.docName, isOwn?{color:'#fff'}:{}]} numberOfLines={1}>{att.fileName}</Text>
                <Text style={[styles.docMeta, isOwn?{color:'rgba(255,255,255,0.7)'}:{}]}>{att.mimeType} · {(att.fileSize/1024).toFixed(1)} KB</Text>
              </View>
              <Text style={styles.docDownload}>⬇</Text>
            </TouchableOpacity>
          ))}
          <Text style={[styles.messageText, isOwn ? styles.ownText : styles.otherText]}>
            {item.text}
          </Text>
          {nativeTranslation && (
            <View style={[styles.translationBlock, isOwn ? styles.ownTranslationBlock : styles.otherTranslationBlock]}>
              <Text style={[styles.translationLabel, isOwn ? styles.ownLabel : styles.otherLabel]}>
                {s.chat.inYourLang}
              </Text>
              <Text style={[styles.translatedText, isOwn ? styles.ownTranslated : styles.otherTranslated]}>
                {nativeTranslation}
              </Text>
            </View>
          )}
          {!isOwn && !nativeTranslation && (
            <TouchableOpacity
              onPress={() => handleTranslate(item.id)}
              disabled={translatingId === item.id}
              style={styles.translateButton}
            >
              <Text style={styles.translateButtonText}>
                {translatingId === item.id ? s.chat.translating : s.chat.translateBtn}
              </Text>
            </TouchableOpacity>
          )}
          {(() => {
            const job = (grammarJobs as any)[item.id];
            const busy = job?.status === 'queued' || job?.status === 'processing' || grammarLoadingId === item.id;
            if (job?.status === 'done' && job?.analysis) {
              return (
                <TouchableOpacity
                  onPress={() => setGrammarModal({ message: item, analysis: job.analysis, provider: job.providerUsed })}
                  style={styles.grammarBadge}
                  testID={`grammar-badge-${item.id}`}>
                  <Text style={styles.grammarBadgeText}>
                    {job.analysis.difficulty ? t('chat.grammarBadge', { level: job.analysis.difficulty }) : s.chat.grammarBadgeReady}
                  </Text>
                </TouchableOpacity>
              );
            }
            if (busy) {
              return (
                <View style={styles.grammarBadgeMuted} testID={`grammar-pending-${item.id}`}>
                  <Text style={styles.grammarBadgeMutedText}>{s.chat.grammarAnalyzing}</Text>
                </View>
              );
            }
            return (
              <TouchableOpacity
                onPress={() => handleGrammarAnalyze(item)}
                style={styles.grammarRequest}
                testID={`grammar-request-${item.id}`}>
                <Text style={styles.grammarRequestText}>{s.chat.grammarRequest}</Text>
              </TouchableOpacity>
            );
          })()}
          {isOwn && (() => {
            const receipts: any[] = (item as any).receipts || [];
            let status: 'sent'|'delivered'|'read' = 'sent';
            if (receipts.length) {
              if (receipts.some((r: any) => r.status === 'read')) status = 'read';
              else if (receipts.some((r: any) => r.status === 'delivered')) status = 'delivered';
            }
            const color = status === 'read' ? '#0ea5e9' : status === 'delivered' ? 'rgba(255,255,255,0.7)' : 'rgba(255,255,255,0.5)';
            const icon = status === 'sent' ? '✓' : '✓✓';
            return (
              <View style={styles.checkRow}>
                <Text style={[styles.checkmark, { color }]}>{icon}</Text>
              </View>
            );
          })()}
        </View>
        <Text style={[styles.messageTime, isOwn ? styles.ownTime : styles.otherTime]}>
          {new Date(item.timestamp).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
        </Text>
      </View>
      </TouchableOpacity>
    );
  };

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color={COLOR.primary} />
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={90}>
      {pinned.length>0 && (
        <TouchableOpacity onPress={()=>setPinnedOpen(!pinnedOpen)} style={{backgroundColor:'#fef3c7', padding:10, flexDirection:'row', justifyContent:'space-between', alignItems:'center', borderBottomWidth:1, borderBottomColor:'#fde68a'}}>
          <Text style={{fontSize:13, fontWeight:'600', color:'#92400e'}}>{t('chat.pinnedCount', { count: pinned.length })} {pinnedOpen?'▴':'▾'}</Text>
          <Text style={{fontSize:11, color:'#92400e', flex:1, textAlign:'right', marginLeft:8}} numberOfLines={1}>{pinned[0]?.message?.text}</Text>
        </TouchableOpacity>
      )}
      {pinnedOpen && pinned.map((p:any)=>(
        <View key={p.message.id} style={{backgroundColor:'#fef3c7', padding:8, flexDirection:'row', justifyContent:'space-between', alignItems:'center', borderBottomWidth:1, borderBottomColor:'#fde68a'}}>
          <Text style={{flex:1, fontSize:13}} numberOfLines={1}>{p.message.text}</Text>
          <TouchableOpacity onPress={()=>{apiService.unpinMessage(chatId,p.message.id).catch(()=>{}); setPinned(prev=>prev.filter((x:any)=>x.message.id!==p.message.id))}}><Text style={{color:'#b45309', fontSize:12, marginLeft:8}}>{s.chat.unpin}</Text></TouchableOpacity>
        </View>
      ))}
      <FlatList
        ref={flatListRef}
        data={messages}
        renderItem={renderMessage}
        keyExtractor={(item) => item.id}
        inverted
        contentContainerStyle={styles.messageList}
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyText}>{s.chat.emptyTitle}</Text>
            <Text style={styles.emptySubtext}>{s.chat.emptySub}</Text>
          </View>
        }
      />
      {typing && (
        <View style={styles.typingIndicator}>
          <Text style={styles.typingText}>{s.chat.typing}</Text>
        </View>
      )}
      {realTalkEnabled && <RealTalkNudge chatId={chatId} onSendToInput={setInputText} />}
      {replyTo && (
        <View style={{backgroundColor:'#e0e7ff', padding:8, flexDirection:'row', justifyContent:'space-between', alignItems:'center', borderTopWidth:1, borderTopColor:COLOR.outlineVariant}}>
          <View style={{borderLeftWidth:2, borderLeftColor:COLOR.primary, paddingLeft:8, flex:1}}>
            <Text style={{fontSize:12, fontWeight:'600', color:COLOR.primary}}>Replying to {replyTo.sender?.displayName || 'Unknown'}</Text>
            <Text style={{fontSize:13, color:COLOR.onSurfaceVariant}} numberOfLines={1}>{replyTo.text}</Text>
          </View>
          <TouchableOpacity onPress={()=>setReplyTo(null)} style={{padding:8}}><Text style={{fontSize:16}}>✕</Text></TouchableOpacity>
        </View>
      )}
      <View style={styles.inputArea}>
      {translateTypeEnabled && (
      <>
        <View style={styles.toggleRow}>
          <TouchableOpacity
            style={styles.toggleLabel}
            onPress={() => setTranslateAsType((prev) => !prev)}
            accessibilityRole="switch"
            accessibilityState={{ checked: translateAsType }}>
            <Text style={styles.toggleIcon}>🌐</Text>
            <Text style={[styles.toggleText, translateAsType && styles.toggleTextActive]}>
              {s.chat.translateToggle}
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.switch, translateAsType && styles.switchOn]}
            onPress={() => setTranslateAsType((prev) => !prev)}>
            <View style={[styles.switchThumb, translateAsType && styles.switchThumbOn]} />
          </TouchableOpacity>
        </View>

        {translateAsType && inputText.trim().length > 0 && (
          <View style={styles.livePreview}>
            <Text style={styles.livePreviewLabel}>
              {t('chat.livePreview', { lang: targetLang })}
            </Text>
            <Text style={styles.livePreviewText}>{inputText}</Text>
          </View>
        )}
      </>
      )}

        {uploading && <View style={{padding:6, alignItems:'center'}}><ActivityIndicator size="small" color={COLOR.primary} /><Text style={{fontSize:11, color:COLOR.onSurfaceVariant}}>{s.chat.uploading}</Text></View>}
        <View style={styles.inputRow}>
          {(mediaEnabled || docsEnabled) && (
            <TouchableOpacity style={styles.iconButton} onPress={handleAttach} accessibilityLabel={s.chat.attachDoc}>
              <Text style={styles.iconButtonText}>📎</Text>
            </TouchableOpacity>
          )}
          {locationEnabled && (
            <TouchableOpacity style={styles.iconButton} onPress={handleShareLocation} accessibilityLabel={s.chat.shareLocation}>
              <Text style={styles.iconButtonText}>📍</Text>
            </TouchableOpacity>
          )}
          <TextInput
            style={styles.input}
            value={inputText}
            onChangeText={(text) => {
              setInputText(text);
              handleTyping();
            }}
            placeholder={s.chat.composerPh}
            placeholderTextColor={COLOR.onSurfaceVariant}
            multiline
            maxLength={1000}
          />
          {voiceEnabled && (
            <TouchableOpacity style={styles.iconButton}>
              <Text style={styles.iconButtonText}>🎙</Text>
            </TouchableOpacity>
          )}
          <TouchableOpacity
            style={[styles.sendButton, !inputText.trim() && styles.sendButtonDisabled]}
            onPress={handleSend}
            disabled={!inputText.trim()}>
            <Text style={styles.sendButtonText}>➤</Text>
          </TouchableOpacity>
        </View>
      </View>

      {/* Sparky FAB */}
      <TouchableOpacity
        style={styles.sparkyFab}
        onPress={openSparkySheet}
        accessibilityLabel="Ask Sparky">
        <Text style={styles.sparkyFabIcon}>🤖</Text>
        <View style={styles.sparkyDot} />
      </TouchableOpacity>

      {/* Deep Dive bottom sheet */}
      <Modal
        visible={deepDiveVisible}
        transparent
        animationType="slide"
        onRequestClose={() => setDeepDiveVisible(false)}>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}
          style={styles.sheetOverlay}>
          <TouchableOpacity
            style={styles.sheetBackdrop}
            activeOpacity={1}
            onPress={() => setDeepDiveVisible(false)}
          />
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <View style={styles.sheetHeader}>
              <View style={styles.sheetTitleRow}>
                <Text style={styles.sheetTitleIcon}>✨</Text>
                <Text style={styles.sheetTitle}>{s.chat.sparkyTitle}</Text>
              </View>
              <TouchableOpacity onPress={() => setDeepDiveVisible(false)}>
                <Text style={styles.sheetClose}>✕</Text>
              </TouchableOpacity>
            </View>
            <ScrollView
              style={styles.sheetBodyScroll}
              contentContainerStyle={styles.sheetBody}
              keyboardShouldPersistTaps="handled"
              testID="sparky-scroll">
              <View style={styles.sheetIntro}>
                <Text style={styles.sheetIntroTitle}>{s.chat.sparkyInsight}</Text>
                <Text style={styles.sheetIntroText}>{s.chat.sparkyInsightSub}</Text>
              </View>
              <View style={styles.sheetTutorBubble}>
                <Text style={styles.sheetTutorText}>
                  {s.chat.sparkyIntro}
                </Text>
              </View>
              {sparkyMessages.map((m, i) => (
                <View key={i} testID={m.role === 'user' ? 'sparky-user-message' : 'sparky-assistant-message'} style={[m.role === 'user' ? { alignSelf: 'flex-end', backgroundColor: COLOR.primary, padding: 10, borderRadius: 12, maxWidth: '85%', marginTop: 8 } : styles.sheetTutorBubble, { marginTop: 8 }]}>
                  <Text style={m.role === 'user' ? { color: '#fff' } : styles.sheetTutorText}>{m.content}</Text>
                </View>
              ))}
              {sparkyLoading && !(
                sparkyJobRef.current != null &&
                sparkyMessages.length > 0 &&
                (sparkyMessages[sparkyMessages.length - 1] as any)?.jobId === sparkyJobRef.current
              ) && <Text testID="sparky-loading" style={styles.typingText}>{s.chat.sparkyTyping}</Text>}
              {sparkyError ? <Text testID="sparky-error" style={{ color: 'red', fontSize: 12 }}>{sparkyError}</Text> : null}
            </ScrollView>
            <View style={styles.sheetInputRow}>
              <TextInput
                style={styles.sheetInput}
                value={sparkyInput}
                onChangeText={setSparkyInput}
                placeholder={s.chat.sparkyInputPh}
                placeholderTextColor={COLOR.onSurfaceVariant}
                multiline
                testID="sparky-input"
              />
              <TouchableOpacity
                testID="sparky-send"
                accessibilityLabel={s.chat.sparkySendA11y}
                style={[styles.sheetSend, (!sparkyInput.trim() || sparkyLoading) && styles.sendButtonDisabled]}
                disabled={!sparkyInput.trim() || sparkyLoading}
                onPress={async () => {
                  const query = sparkyInput.trim();
                  if (!query || sparkyLoading) return;
                  // Sparky context = the last 3 non-empty messages (newest
                  // first in this list), so follow-ups like "analyze the last
                  // message" resolve against real conversation context.
                  const contextText =
                    messages.filter((m: any) => m?.text?.trim()).slice(0, 3).map((m: any) =>
                      `${m.senderId === currentUser?.id ? 'Me' : (m.sender?.displayName || 'Them')}: ${m.text}`
                    ).join('\n') || query;
                  const lang = currentUser?.targetLanguages?.[0] || 'es';
                  const nativeLang = currentUser?.nativeLanguage || 'en';
                  setSparkyInput('');
                  setSparkyError('');
                  setSparkyMessages(prev => [...prev, { role: 'user', content: query }]);
                  setSparkyLoading(true);
                  // Phase 1: async job first — 202 instantly, answer via WS.
                  // Falls back to the sync path only if the queue is down.
                  try {
                    const asked: any = await (apiService as any).sparkyAsk({
                      context: contextText,
                      query,
                      language: lang,
                      nativeLanguage: nativeLang,
                      chatId,
                    });
                    if (asked?.jobId) {
                      sparkyJobRef.current = asked.jobId;
                      await storage.setItem(SPARKY_PENDING_KEY, JSON.stringify({ jobId: asked.jobId, chatId })).catch(() => {});
                      return;
                    }
                    throw new Error('Sparky queue did not return a job');
                  } catch (asyncErr: any) {
                    // Queue unavailable — fall back to the sync path (60s
                    // budget). If that fails too, surface its error.
                    try {
                      const res: any = await (apiService as any).grammarLearn(contextText, lang, nativeLang, 'custom', query);
                      const content = res?.content || 'Done';
                      setSparkyMessages(prev => [...prev, { role: 'assistant', content }]);
                  } catch (e: any) {
                    const msg = apiErrorMessage(e, t('chat.sparkyFailB'));
                    setSparkyError(msg);
                    setSparkyMessages(prev => [...prev, { role: 'assistant', content: msg }]);
                  } finally {
                    setSparkyLoading(false);
                  }
                  }
                }}>
                <Text style={styles.sheetSendText}>➤</Text>
              </TouchableOpacity>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>
      <Modal visible={!!grammarModal} transparent animationType="slide" onRequestClose={()=>setGrammarModal(null)}>
        <View style={styles.sheetOverlay}><TouchableOpacity style={styles.sheetBackdrop} activeOpacity={1} onPress={()=>setGrammarModal(null)} />
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <View style={styles.sheetHeader}>
              <View style={styles.sheetTitleRow}>
                <Text style={styles.sheetTitleIcon}>✨</Text>
                <Text style={styles.sheetTitle}>{s.chat.grammarModalTitle}{grammarModal?.analysis?.difficulty ? ` · ${grammarModal.analysis.difficulty}` : ''}</Text>
              </View>
              <TouchableOpacity onPress={()=>setGrammarModal(null)}><Text style={styles.sheetClose}>✕</Text></TouchableOpacity>
            </View>
            <View style={styles.sheetBody}>
              <Text style={styles.grammarBodyText} numberOfLines={2}>{grammarModal?.message?.text}</Text>
              {!!grammarModal?.analysis?.summary && (
                <><Text style={styles.grammarSectionTitle}>{s.chat.grammarSummary}</Text><Text style={styles.grammarBodyText}>{grammarModal.analysis.summary}</Text></>
              )}
              {!!grammarModal?.analysis?.sentenceStructure && (
                <><Text style={styles.grammarSectionTitle}>{s.chat.grammarStructure}</Text><Text style={styles.grammarBodyText}>{grammarModal.analysis.sentenceStructure}</Text></>
              )}
              {(grammarModal?.analysis?.keyPhrases || []).length > 0 && (
                <><Text style={styles.grammarSectionTitle}>{s.chat.grammarKeyPhrases}</Text>{grammarModal.analysis.keyPhrases.slice(0, 4).map((k: any, i: number) => (
                  <Text key={i} style={styles.grammarBodyText}>• {k.phrase} — {k.translation}</Text>
                ))}</>
              )}
              {(grammarModal?.analysis?.grammarNotes || []).length > 0 && (
                <><Text style={styles.grammarSectionTitle}>{s.chat.grammarNotes}</Text>{grammarModal.analysis.grammarNotes.slice(0, 3).map((n: any, i: number) => (
                  <Text key={i} style={styles.grammarBodyText}>• {n.title}: {n.explanation}</Text>
                ))}</>
              )}
              {(grammarModal?.analysis?.detailedBreakdown || []).length > 0 && (
                <><Text style={styles.grammarSectionTitle}>{s.chat.grammarBreakdown}</Text>{grammarModal.analysis.detailedBreakdown.slice(0, 6).map((b: any, i: number) => (
                  <Text key={i} style={styles.grammarBodyText}>• {b.text} — {b.translation || b.role || ''}</Text>
                ))}</>
              )}
              {!!grammarModal?.provider && (
                <Text style={[styles.grammarBodyText, { marginTop: 12, fontSize: 11 }]}>via {grammarModal.provider}</Text>
              )}
            </View>
          </View>
        </View>
      </Modal>
      <Modal visible={!!actionMsg} transparent animationType="fade" onRequestClose={()=>setActionMsg(null)}>
        <View style={styles.sheetOverlay}><TouchableOpacity style={styles.sheetBackdrop} activeOpacity={1} onPress={()=>setActionMsg(null)} />
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <Text style={{fontSize:13, color:COLOR.onSurfaceVariant, marginBottom:8}} numberOfLines={2}>{actionMsg?.text}</Text>
            <TouchableOpacity style={styles.actionRow} onPress={()=>{setReplyTo(actionMsg); setActionMsg(null)}}><Text style={styles.actionText}>{s.chat.reply}</Text></TouchableOpacity>
            <TouchableOpacity style={styles.actionRow} onPress={async()=>{ if(!actionMsg) return; const c=actionMsg; setActionMsg(null); try{ const chats=await apiService.getChats(); setForwardChats(chats.filter((x:any)=>x.id!==chatId)); setForwardMsg(c)}catch{}}}><Text style={styles.actionText}>{s.chat.forward}</Text></TouchableOpacity>
            <TouchableOpacity style={styles.actionRow} onPress={async()=>{ if(!actionMsg) return; const m=actionMsg; const isPinned=pinned.some((p:any)=>p.message.id===m.id); setActionMsg(null); try{ if(isPinned) await apiService.unpinMessage(chatId,m.id); else await apiService.pinMessage(chatId,m.id); const p=await apiService.getPinnedMessages(chatId); setPinned(p ?? [])}catch{}}}><Text style={styles.actionText}>📌 {actionMsg && pinned.some((p:any)=>p.message.id===actionMsg.id) ? s.chat.unpinAction : s.chat.pin}</Text></TouchableOpacity>
            {actionMsg?.senderId===currentUser?.id && <TouchableOpacity style={styles.actionRow} onPress={async()=>{ if(!actionMsg) return; const id=actionMsg.id; setActionMsg(null); try{ await apiService.deleteMessage(chatId,id); setMessages(prev=>prev.filter(m=>m.id!==id))}catch{}}}><Text style={[styles.actionText,{color:'red'}]}>{s.chat.delete}</Text></TouchableOpacity>}
            {actionMsg && actionMsg.senderId !== currentUser?.id && <TouchableOpacity style={styles.actionRow} onPress={()=>{ const m=actionMsg; setActionMsg(null); if(m) openReport({ type:'message', messageId: m.id, chatId, userId: m.senderId }) }}><Text style={styles.actionText}>{s.chat.reportMsg}</Text></TouchableOpacity>}
            <TouchableOpacity style={[styles.actionRow,{marginTop:8}]} onPress={()=>setActionMsg(null)}><Text style={[styles.actionText,{textAlign:'center', color:COLOR.onSurfaceVariant}]}>{s.chat.cancelAction}</Text></TouchableOpacity>
          </View>
        </View>
      </Modal>
      <Modal visible={reportVisible} transparent animationType="fade" onRequestClose={()=>setReportVisible(false)}>
        <View style={styles.sheetOverlay}><TouchableOpacity style={styles.sheetBackdrop} activeOpacity={1} onPress={()=>setReportVisible(false)} />
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <View style={styles.sheetHeader}><Text style={styles.sheetTitle}>{reportTarget?.type === 'message' ? s.chat.reportTitleMsg : s.chat.reportTitleUser}</Text><TouchableOpacity onPress={()=>setReportVisible(false)}><Text style={styles.sheetClose}>✕</Text></TouchableOpacity></View>
            {(['spam','harassment','inappropriate','scam','other'] as const).map(r=>{
              const label = { spam: s.chat.reasonSpam, harassment: s.chat.reasonHarassment, inappropriate: s.chat.reasonInappropriate, scam: s.chat.reasonScam, other: s.chat.reasonOther }[r];
              return (
                <TouchableOpacity key={r} style={[styles.actionRow, reportReason===r && {backgroundColor: COLOR.primaryContainer}]} onPress={()=>setReportReason(r)}>
                  <Text style={[styles.actionText, reportReason===r && {color: COLOR.onPrimaryContainer, fontWeight:'700'}]}>{label}</Text>
                </TouchableOpacity>
              );
            })}
            <TouchableOpacity style={[styles.actionRow,{backgroundColor: COLOR.error, borderRadius:8, marginTop:8, justifyContent:'center'}]} onPress={submitReport}><Text style={[styles.actionText,{color:'#fff', textAlign:'center'}]}>{s.chat.submitReport}</Text></TouchableOpacity>
          </View>
        </View>
      </Modal>
      <Modal visible={!!forwardMsg} transparent animationType="slide" onRequestClose={()=>setForwardMsg(null)}>
        <View style={styles.sheetOverlay}><TouchableOpacity style={styles.sheetBackdrop} activeOpacity={1} onPress={()=>setForwardMsg(null)} />
          <View style={[styles.sheet,{maxHeight:'70%'}]}>
            <View style={styles.sheetHandle} />
            <View style={styles.sheetHeader}><Text style={styles.sheetTitle}>{s.chat.forwardTo}</Text><TouchableOpacity onPress={()=>setForwardMsg(null)}><Text style={styles.sheetClose}>✕</Text></TouchableOpacity></View>
            <Text style={{fontSize:12, color:COLOR.onSurfaceVariant, marginBottom:8}} numberOfLines={2}>"{forwardMsg?.text.slice(0,80)}"</Text>
            {forwardChats.length===0 ? <Text style={{textAlign:'center', color:COLOR.onSurfaceVariant, padding:20}}>{s.chat.noOtherChats}</Text> : forwardChats.map((c:any)=>{ const name=c.type==='group'?(c.name||s.chat.groupFallback):(c.participants?.find((p:any)=>p.user)?.user?.displayName||s.chat.chatFallback); return <TouchableOpacity key={c.id} style={styles.actionRow} onPress={async()=>{ if(!forwardMsg) return; try{ await apiService.forwardMessage(chatId, forwardMsg.id, c.id); setForwardMsg(null)}catch{}}}><Text style={styles.actionText}>{name}</Text></TouchableOpacity>})}
          </View>
        </View>
      </Modal>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: COLOR.background,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  messageList: {
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  messageContainer: {
    marginVertical: 6,
    maxWidth: '80%',
  },
  ownMessage: {
    alignSelf: 'flex-end',
    alignItems: 'flex-end',
  },
  otherMessage: {
    alignSelf: 'flex-start',
    alignItems: 'flex-start',
  },
  senderName: {
    fontSize: 12,
    fontWeight: '600',
    color: COLOR.onSurfaceVariant,
    marginBottom: 4,
    marginLeft: 16,
  },
  messageBubble: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    shadowColor: '#000',
    shadowOpacity: 0.05,
    shadowRadius: 4,
    shadowOffset: { width: 0, height: 2 },
    elevation: 1,
  },
  ownBubble: {
    backgroundColor: COLOR.primary,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    borderBottomLeftRadius: 24,
    borderBottomRightRadius: 4,
  },
  otherBubble: {
    backgroundColor: COLOR.surfaceContainerLowest,
    borderColor: COLOR.outlineVariant,
    borderWidth: 1,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    borderBottomLeftRadius: 4,
    borderBottomRightRadius: 24,
  },
  messageText: {
    fontSize: 16,
    lineHeight: 24,
    fontFamily: FONTS.body,
  },
  ownText: {
    color: COLOR.onPrimary,
  },
  otherText: {
    color: COLOR.onSurface,
  },
  translationBlock: {
    marginTop: 8,
    paddingTop: 8,
    borderTopWidth: 1,
  },
  ownTranslationBlock: {
    borderTopColor: 'rgba(255, 255, 255, 0.3)',
  },
  otherTranslationBlock: {
    borderTopColor: 'rgba(67, 70, 85, 0.3)',
  },
  translationLabel: {
    fontSize: 11,
    marginBottom: 2,
  },
  ownLabel: {
    color: 'rgba(255, 255, 255, 0.75)',
  },
  otherLabel: {
    color: COLOR.onSurfaceVariant,
  },
  translatedText: {
    fontSize: 14,
    lineHeight: 20,
    fontStyle: 'italic',
  },
  ownTranslated: {
    color: COLOR.onPrimary,
  },
  otherTranslated: {
    color: COLOR.onSurfaceVariant,
  },
  translateButton: {
    marginTop: 8,
    alignSelf: 'flex-start',
  },
  translateButtonText: {
    fontSize: 12,
    color: COLOR.primary,
    fontWeight: '600',
  },
  grammarBadge: {
    marginTop: 8,
    alignSelf: 'flex-start',
    backgroundColor: '#eef2ff',
    borderRadius: 12,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  grammarBadgeText: {
    fontSize: 12,
    color: COLOR.primary,
    fontWeight: '700',
  },
  grammarBadgeMuted: {
    marginTop: 8,
    alignSelf: 'flex-start',
  },
  grammarBadgeMutedText: {
    fontSize: 12,
    color: COLOR.onSurfaceVariant,
    fontStyle: 'italic',
  },
  grammarRequest: {
    marginTop: 8,
    alignSelf: 'flex-start',
  },
  grammarRequestText: {
    fontSize: 12,
    color: COLOR.onSurfaceVariant,
    fontWeight: '600',
  },
  grammarSectionTitle: {
    fontSize: 13,
    fontWeight: '700',
    color: COLOR.onSurface,
    marginTop: 12,
    marginBottom: 4,
  },
  grammarBodyText: {
    fontSize: 13,
    color: COLOR.onSurfaceVariant,
    lineHeight: 18,
  },
  checkRow: {
    alignItems: 'flex-end',
    marginTop: 2,
  },
  checkmark: {
    fontSize: 12,
    color: COLOR.onPrimary,
    opacity: 0.7,
  },
  messageTime: {
    fontSize: 11,
    marginTop: 4,
  },
  ownTime: {
    color: COLOR.onSurfaceVariant,
    marginRight: 4,
  },
  otherTime: {
    color: COLOR.onSurfaceVariant,
    marginLeft: 4,
  },
  typingIndicator: {
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  typingText: {
    fontSize: 12,
    color: COLOR.onSurfaceVariant,
    fontStyle: 'italic',
  },
  inputArea: {
    backgroundColor: COLOR.surface,
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
    paddingHorizontal: 4,
  },
  toggleLabel: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  toggleIcon: {
    fontSize: 18,
    marginRight: 6,
  },
  toggleText: {
    fontSize: 13,
    fontWeight: '600',
    color: COLOR.onSurfaceVariant,
  },
  toggleTextActive: {
    color: COLOR.primary,
  },
  switch: {
    width: 40,
    height: 20,
    borderRadius: 10,
    backgroundColor: COLOR.surfaceContainerLow,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    justifyContent: 'center',
  },
  switchOn: {
    backgroundColor: COLOR.primary,
    borderColor: COLOR.primary,
  },
  switchThumb: {
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: '#fff',
    shadowColor: '#000',
    shadowOpacity: 0.2,
    shadowRadius: 2,
    shadowOffset: { width: 0, height: 1 },
    marginLeft: 2,
  },
  switchThumbOn: {
    marginLeft: 22,
  },
  livePreview: {
    marginBottom: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(107, 56, 212, 0.3)',
    backgroundColor: 'rgba(233, 221, 255, 0.25)',
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  livePreviewLabel: {
    fontSize: 11,
    fontWeight: '600',
    color: COLOR.secondary,
    marginBottom: 2,
  },
  livePreviewText: {
    fontSize: 14,
    fontStyle: 'italic',
    color: COLOR.onSurfaceVariant,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
  },
  iconButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  iconButtonText: {
    fontSize: 20,
    color: COLOR.primary,
  },
  input: {
    flex: 1,
    backgroundColor: COLOR.surfaceContainerLow,
    borderRadius: 24,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    paddingHorizontal: 16,
    paddingVertical: 10,
    marginHorizontal: 6,
    maxHeight: 100,
    fontSize: 16,
    color: COLOR.onSurface,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: COLOR.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendButtonDisabled: {
    opacity: 0.4,
  },
  sendButtonText: {
    color: COLOR.onPrimary,
    fontSize: 18,
    lineHeight: 20,
  },
  sparkyFab: {
    position: 'absolute',
    bottom: 96,
    right: 16,
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: COLOR.secondary,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: COLOR.secondary,
    shadowOpacity: 0.35,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 4 },
    elevation: 6,
  },
  sparkyFabIcon: {
    fontSize: 28,
  },
  sparkyDot: {
    position: 'absolute',
    top: 4,
    right: 4,
    width: 12,
    height: 12,
    borderRadius: 6,
    backgroundColor: COLOR.tertiary,
    borderWidth: 2,
    borderColor: COLOR.surface,
  },
  sheetOverlay: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  sheetBackdrop: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(11, 28, 48, 0.4)',
  },
  sheet: {
    backgroundColor: COLOR.surface,
    borderTopLeftRadius: 32,
    borderTopRightRadius: 32,
    paddingHorizontal: 16,
    paddingBottom: 24,
    maxHeight: '85%',
  },
  sheetHandle: {
    alignSelf: 'center',
    width: 48,
    height: 5,
    borderRadius: 3,
    backgroundColor: COLOR.outlineVariant,
    marginTop: 10,
    marginBottom: 8,
  },
  sheetHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: COLOR.outlineVariant,
  },
  sheetTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  sheetTitleIcon: {
    fontSize: 18,
    marginRight: 8,
  },
  sheetTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
  },
  sheetClose: {
    fontSize: 18,
    color: COLOR.onSurfaceVariant,
    padding: 4,
  },
  sheetBody: {
    paddingVertical: 16,
  },
  // Scrollable message area: shrinks inside the sheet's maxHeight so long
  // answers scroll instead of pushing the input row off-screen (the
  // disappearing-input bug). The input row stays pinned below.
  sheetBodyScroll: {
    flexGrow: 0,
    flexShrink: 1,
  },
  sheetIntro: {
    alignItems: 'center',
    marginBottom: 16,
  },
  sheetIntroTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: COLOR.onSurface,
    fontFamily: FONTS.headline,
    marginBottom: 4,
  },
  sheetIntroText: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
    fontFamily: FONTS.body,
  },
  sheetTutorBubble: {
    alignSelf: 'flex-start',
    maxWidth: '85%',
    backgroundColor: COLOR.surfaceContainerLow,
    borderRadius: 16,
    borderBottomLeftRadius: 4,
    padding: 12,
  },
  sheetTutorText: {
    fontSize: 14,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
    lineHeight: 20,
  },
  sheetInputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    borderTopWidth: 1,
    borderTopColor: COLOR.outlineVariant,
    paddingTop: 12,
  },
  sheetInput: {
    flex: 1,
    backgroundColor: COLOR.surfaceContainerLowest,
    borderWidth: 1,
    borderColor: COLOR.outlineVariant,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 10,
    marginRight: 8,
    maxHeight: 100,
    fontSize: 14,
    color: COLOR.onSurface,
    fontFamily: FONTS.body,
  },
  sheetSend: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: COLOR.secondary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sheetSendText: {
    color: '#fff',
    fontSize: 18,
    lineHeight: 20,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingTop: 100,
  },
  emptyText: {
    fontSize: 18,
    fontWeight: '600',
    color: COLOR.onSurface,
    marginBottom: 8,
  },
  emptySubtext: {
    fontSize: 14,
    color: COLOR.onSurfaceVariant,
  },
  forwardLabel:{fontSize:11, marginBottom:2, fontStyle:'italic'},
  replyQuote:{borderLeftWidth:2, paddingLeft:6, marginBottom:6, borderRadius:4, paddingVertical:2},
  replyQuoteOwn:{borderLeftColor:'rgba(255,255,255,0.6)', backgroundColor:'rgba(255,255,255,0.15)'},
  replyQuoteOther:{borderLeftColor:COLOR.primary, backgroundColor:COLOR.surfaceContainerHigh},
  replyQuoteName:{fontSize:11, fontWeight:'600', color:COLOR.primary},
  replyQuoteText:{fontSize:12, color:COLOR.onSurfaceVariant},
  actionRow:{paddingVertical:14, borderBottomWidth:1, borderBottomColor:COLOR.outlineVariant},
  actionText:{fontSize:16, fontWeight:'500', color:COLOR.onSurface},
  docCard:{flexDirection:'row', alignItems:'center', gap:8, padding:10, borderRadius:12, borderWidth:1, marginBottom:6},
  docCardOwn:{backgroundColor:'rgba(255,255,255,0.15)', borderColor:'rgba(255,255,255,0.2)'},
  docCardOther:{backgroundColor:COLOR.surfaceContainerHigh, borderColor:COLOR.outlineVariant},
  docIcon:{fontSize:20},
  docName:{fontSize:13, fontWeight:'600', color:COLOR.onSurface},
  docMeta:{fontSize:11, color:COLOR.onSurfaceVariant},
  docDownload:{fontSize:16, color:COLOR.primary},
});
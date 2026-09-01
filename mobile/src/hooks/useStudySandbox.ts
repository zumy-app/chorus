import { useCallback, useState } from 'react';

export type StudySandboxState = {
  prompt: string;
  topic: string;
  activityLog: { id: string; user: string; text: string; at: string }[];
  sharedSandboxState: Record<string, unknown>;
};

const defaultState: StudySandboxState = {
  prompt: 'Describe the most surprising thing that happened on your last trip.',
  topic: 'Real Talk: Travel Stories',
  activityLog: [],
  sharedSandboxState: {},
};

export function useStudySandbox(initial?: Partial<StudySandboxState>) {
  const [state, setState] = useState<StudySandboxState>({ ...defaultState, ...initial });
  const setPrompt = useCallback((prompt: string, topic?: string) => {
    setState((s) => ({ ...s, prompt, topic: topic ?? s.topic }));
  }, []);
  const addActivity = useCallback((user: string, text: string) => {
    setState((s) => ({
      ...s,
      activityLog: [...s.activityLog, { id: `${Date.now()}`, user, text, at: new Date().toISOString() }].slice(-20),
    }));
  }, []);
  const updateShared = useCallback((patch: Record<string, unknown>) => {
    setState((s) => ({ ...s, sharedSandboxState: { ...s.sharedSandboxState, ...patch } }));
  }, []);
  const clear = useCallback(() => setState(defaultState), []);
  return { state, setState, setPrompt, addActivity, updateShared, clear };
}

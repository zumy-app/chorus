import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { moderationAPI } from '../services/api'

interface ReportModalProps {
  targetType: 'user' | 'message'
  targetUserId?: string
  messageId?: string
  chatId?: string
  reportedUserName?: string
  onClose: () => void
}

const REPORT_REASONS = ['spam', 'harassment', 'inappropriate', 'scam', 'other'] as const

export default function ReportModal({
  targetType,
  targetUserId,
  messageId,
  chatId,
  reportedUserName,
  onClose,
}: ReportModalProps) {
  const { t } = useTranslation()
  const [reason, setReason] = useState<string>('spam')
  const [detail, setDetail] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      await moderationAPI.report({
        type: targetType,
        reportedUserId: targetType === 'user' ? targetUserId : undefined,
        messageId: targetType === 'message' ? messageId : undefined,
        chatId,
        reason: detail.trim() ? `${reason}: ${detail.trim()}` : reason,
      })
      setDone(true)
      setTimeout(onClose, 1200)
    } catch (err: any) {
      setError(err?.response?.data?.error || t('report.submitError'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-lg shadow-xl w-full max-w-md max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-lg font-bold text-gray-900">
            {done ? '✓' : '🚩'} {targetType === 'user' ? t('report.reportUser') : t('report.reportMessage')}
            {reportedUserName ? ` — ${reportedUserName}` : ''}
          </h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">×</button>
        </div>

        {done ? (
          <div className="p-6 text-green-700 font-medium">{t('report.thanks')}</div>
        ) : (
          <form onSubmit={handleSubmit} className="p-5 space-y-4">
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">{t('report.reasonLabel')}</label>
              <div className="space-y-2">
                {REPORT_REASONS.map((r) => (
                  <label
                    key={r}
                    className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition ${
                      reason === r ? 'border-indigo-500 bg-indigo-50' : 'border-gray-200 hover:bg-gray-50'
                    }`}
                  >
                    <input
                      type="radio"
                      name="reportReason"
                      value={r}
                      checked={reason === r}
                      onChange={() => setReason(r)}
                      className="text-indigo-600 focus:ring-indigo-500"
                    />
                    <span className="text-sm text-gray-800">{t(`report.reason.${r}`)}</span>
                  </label>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-bold text-gray-700 mb-2">
                {t('report.detailLabel')} <span className="text-gray-400 font-normal">({t('report.optional')})</span>
              </label>
              <textarea
                value={detail}
                onChange={(e) => setDetail(e.target.value)}
                rows={3}
                maxLength={400}
                placeholder={t('report.detailPlaceholder')}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
              />
            </div>

            {error && <div className="px-4 py-3 rounded-lg text-sm bg-red-100 text-red-700">{error}</div>}

            <div className="flex justify-end space-x-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="px-5 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="px-5 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 font-semibold"
              >
                {submitting ? t('report.submitting') : t('report.submit')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
'use client';

import React, { useState } from 'react';
import { X, AlertTriangle, Shield, ArchiveRestore, CheckCircle, Loader2 } from 'lucide-react';

interface RemediationConfirmationModalProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (options: RemediationOptions) => Promise<void>;
    findingId: string | null;
    actionType: 'MASK' | 'DELETE';
}

interface RemediationOptions {
    createRollback: boolean;
    notifyOwner: boolean;
}

export function RemediationConfirmationModal({
    isOpen,
    onClose,
    onConfirm,
    findingId,
    actionType
}: RemediationConfirmationModalProps) {
    const [isProcessing, setIsProcessing] = useState(false);
    const [isSuccess, setIsSuccess] = useState(false);
    const [options, setOptions] = useState<RemediationOptions>({
        createRollback: true,
        notifyOwner: true
    });

    if (!isOpen) return null;

    const handleConfirm = async () => {
        setIsProcessing(true);
        try {
            await onConfirm(options);
            setIsSuccess(true);
            setTimeout(() => {
                setIsSuccess(false);
                setIsProcessing(false);
                onClose();
                window.location.reload();
            }, 2000);
        } catch (error) {
            console.error(error);
            setIsProcessing(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[60]">
            <div className="bg-white rounded-lg shadow-2xl w-full max-w-md border border-slate-200 transform transition-all">
                {isSuccess ? (
                    <div className="p-8 flex flex-col items-center justify-center text-center">
                        <div className="w-16 h-16 bg-green-50 rounded-full flex items-center justify-center mb-4 border border-green-200">
                            <CheckCircle className="w-8 h-8 text-green-600" />
                        </div>
                        <h3 className="text-xl font-bold text-slate-900 mb-2">Action Completed</h3>
                        <p className="text-slate-500">
                            The remediation action has been successfully applied.
                        </p>
                    </div>
                ) : (
                    <>
                        {/* Header */}
                        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-slate-50">
                            <div className="flex items-center gap-3">
                                <div className={`p-2 rounded-lg ${actionType === 'DELETE' ? 'bg-red-50' : 'bg-blue-50'}`}>
                                    <Shield className={`w-5 h-5 ${actionType === 'DELETE' ? 'text-red-600' : 'text-blue-600'}`} />
                                </div>
                                <div>
                                    <h3 className="text-lg font-semibold text-slate-900">
                                        Confirm {actionType === 'MASK' ? 'Masking' : 'Deletion'}
                                    </h3>
                                    <p className="text-xs text-slate-500 font-mono">{findingId}</p>
                                </div>
                            </div>
                            <button
                                onClick={onClose}
                                disabled={isProcessing}
                                className="text-slate-400 hover:text-slate-900 transition-colors"
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        {/* Content */}
                        <div className="p-6 space-y-6">
                            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 flex gap-3">
                                <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0" />
                                <div className="text-sm">
                                    <p className="text-amber-800 font-medium mb-1">Warning: Permanent Action</p>
                                    <p className="text-amber-700">
                                        This will modify the source data. Ensure you have authorization to perform this action.
                                    </p>
                                </div>
                            </div>

                            <div className="space-y-4">
                                <label className="flex items-center justify-between p-3 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 cursor-pointer transition-colors">
                                    <div className="flex items-center gap-3">
                                        <ArchiveRestore className="w-5 h-5 text-purple-600" />
                                        <div>
                                            <div className="text-sm font-medium text-slate-900">Create Rollback Point</div>
                                            <div className="text-xs text-slate-500">Save original value for 30 days</div>
                                        </div>
                                    </div>
                                    <input
                                        type="checkbox"
                                        checked={options.createRollback}
                                        onChange={(e) => setOptions({ ...options, createRollback: e.target.checked })}
                                        className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                                    />
                                </label>

                                <label className="flex items-center justify-between p-3 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 cursor-pointer transition-colors">
                                    <div className="flex items-center gap-3">
                                        <Shield className="w-5 h-5 text-slate-500" />
                                        <div>
                                            <div className="text-sm font-medium text-slate-900">Notify Data Owner</div>
                                            <div className="text-xs text-slate-500">Send email notification</div>
                                        </div>
                                    </div>
                                    <input
                                        type="checkbox"
                                        checked={options.notifyOwner}
                                        onChange={(e) => setOptions({ ...options, notifyOwner: e.target.checked })}
                                        className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                                    />
                                </label>
                            </div>
                        </div>

                        {/* Footer */}
                        <div className="px-6 py-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
                            <button
                                onClick={onClose}
                                disabled={isProcessing}
                                className="px-4 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleConfirm}
                                disabled={isProcessing}
                                className={`
                                    flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white transition-all
                                    ${actionType === 'DELETE' ? 'bg-red-600 hover:bg-red-700' : 'bg-blue-600 hover:bg-blue-700'}
                                    ${isProcessing ? 'opacity-75 cursor-not-allowed' : ''}
                                `}
                            >
                                {isProcessing && <Loader2 className="w-4 h-4 animate-spin" />}
                                {actionType === 'MASK' ? 'Mask Data' : 'Delete Data'}
                            </button>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}

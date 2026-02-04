'use client';

import React, { useState } from 'react';
import { X, AlertTriangle, Shield, ArchiveRestore, CheckCircle, Loader2 } from 'lucide-react';
import { theme } from '@/design-system/theme';

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
                // Trigger a refresh/reload via prop if needed, or rely on parent
                window.location.reload();
            }, 2000);
        } catch (error) {
            console.error(error);
            setIsProcessing(false);
            // Handle error state
        }
    };

    return (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[60]">
            <div className="bg-card rounded-lg shadow-2xl w-full max-w-md border border-border transform transition-all">
                {isSuccess ? (
                    <div className="p-8 flex flex-col items-center justify-center text-center">
                        <div className="w-16 h-16 bg-green-500/10 rounded-full flex items-center justify-center mb-4 border border-green-500/20">
                            <CheckCircle className="w-8 h-8 text-green-500" />
                        </div>
                        <h3 className="text-xl font-bold text-white mb-2">Action Completed</h3>
                        <p className="text-muted-foreground">
                            The remediation action has been successfully applied.
                        </p>
                    </div>
                ) : (
                    <>
                        {/* Header */}
                        <div className="px-6 py-4 border-b border-border flex items-center justify-between bg-secondary">
                            <div className="flex items-center gap-3">
                                <div className={`p-2 rounded-lg ${actionType === 'DELETE' ? 'bg-red-500/10' : 'bg-blue-500/10'}`}>
                                    <Shield className={`w-5 h-5 ${actionType === 'DELETE' ? 'text-red-400' : 'text-blue-400'}`} />
                                </div>
                                <div>
                                    <h3 className="text-lg font-semibold text-white">
                                        Confirm {actionType === 'MASK' ? 'Masking' : 'Deletion'}
                                    </h3>
                                    <p className="text-xs text-muted-foreground font-mono">{findingId}</p>
                                </div>
                            </div>
                            <button
                                onClick={onClose}
                                disabled={isProcessing}
                                className="text-muted-foreground hover:text-white transition-colors"
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        {/* Content */}
                        <div className="p-6 space-y-6">
                            <div className="bg-amber-500/10 border border-amber-500/20 rounded-lg p-4 flex gap-3">
                                <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0" />
                                <div className="text-sm">
                                    <p className="text-amber-200 font-medium mb-1">Warning: Permanent Action</p>
                                    <p className="text-amber-200/70">
                                        This will modify the source data. Ensure you have authorization to perform this action.
                                    </p>
                                </div>
                            </div>

                            <div className="space-y-4">
                                <label className="flex items-center justify-between p-3 rounded-lg border border-border bg-muted hover:bg-secondary cursor-pointer transition-colors">
                                    <div className="flex items-center gap-3">
                                        <ArchiveRestore className="w-5 h-5 text-purple-400" />
                                        <div>
                                            <div className="text-sm font-medium text-white">Create Rollback Point</div>
                                            <div className="text-xs text-muted-foreground">Save original value for 30 days</div>
                                        </div>
                                    </div>
                                    <input
                                        type="checkbox"
                                        checked={options.createRollback}
                                        onChange={(e) => setOptions({ ...options, createRollback: e.target.checked })}
                                        className="w-4 h-4 rounded border-border bg-accent text-blue-500 focus:ring-offset-slate-900"
                                    />
                                </label>

                                <label className="flex items-center justify-between p-3 rounded-lg border border-border bg-muted hover:bg-secondary cursor-pointer transition-colors">
                                    <div className="flex items-center gap-3">
                                        <Shield className="w-5 h-5 text-muted-foreground" />
                                        <div>
                                            <div className="text-sm font-medium text-white">Notify Data Owner</div>
                                            <div className="text-xs text-muted-foreground">Send email notification</div>
                                        </div>
                                    </div>
                                    <input
                                        type="checkbox"
                                        checked={options.notifyOwner}
                                        onChange={(e) => setOptions({ ...options, notifyOwner: e.target.checked })}
                                        className="w-4 h-4 rounded border-border bg-accent text-blue-500 focus:ring-offset-slate-900"
                                    />
                                </label>
                            </div>
                        </div>

                        {/* Footer */}
                        <div className="px-6 py-4 border-t border-border bg-muted flex justify-end gap-3">
                            <button
                                onClick={onClose}
                                disabled={isProcessing}
                                className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-white transition-colors"
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

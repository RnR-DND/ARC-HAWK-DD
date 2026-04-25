import { Skeleton } from '@/components/ui/skeleton';

export default function Loading() {
    return (
        <div className="p-8 space-y-6">
            <div className="space-y-2">
                <Skeleton className="h-8 w-56" />
                <Skeleton className="h-4 w-80" />
            </div>
            <div className="border border-slate-200 rounded-xl overflow-hidden">
                <div className="p-6 space-y-3">
                    {[...Array(6)].map((_, i) => (
                        <div key={i} className="flex gap-4">
                            <div className="animate-pulse rounded-md bg-slate-100 h-12 flex-1" />
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}

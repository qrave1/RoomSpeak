// Audio Activity Detection
export class AudioActivityDetector {
    constructor(stream, onSpeakingChange) {
        this.stream = stream;
        this.onSpeakingChange = onSpeakingChange;
        this.isSpeaking = false;
        this.audioContext = null;
        this.analyser = null;
        this.dataArray = null;
        this.animationFrame = null;
        this.threshold = 0.1; // Порог для определения речи
        this.speakingTimeout = null;
        
        this.init();
    }
    
    init() {
        if (!this.stream) return;
        
        try {
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const source = this.audioContext.createMediaStreamSource(this.stream);
            this.analyser = this.audioContext.createAnalyser();
            
            this.analyser.fftSize = 256;
            this.analyser.smoothingTimeConstant = 0.8;
            
            source.connect(this.analyser);
            
            const bufferLength = this.analyser.frequencyBinCount;
            this.dataArray = new Uint8Array(bufferLength);
            
            this.startDetection();
        } catch (error) {
            console.error('Error initializing audio activity detector:', error);
        }
    }
    
    startDetection() {
        const detect = () => {
            this.analyser.getByteFrequencyData(this.dataArray);
            
            // Вычисляем средний уровень звука
            let sum = 0;
            for (let i = 0; i < this.dataArray.length; i++) {
                sum += this.dataArray[i];
            }
            const average = sum / this.dataArray.length / 255;
            
            const wasSpeaking = this.isSpeaking;
            this.isSpeaking = average > this.threshold;
            
            // Если статус изменился, уведомляем
            if (wasSpeaking !== this.isSpeaking) {
                this.onSpeakingChange(this.isSpeaking);
                
                // Устанавливаем таймаут для остановки индикатора "говорит"
                if (this.isSpeaking) {
                    clearTimeout(this.speakingTimeout);
                } else {
                    this.speakingTimeout = setTimeout(() => {
                        this.onSpeakingChange(false);
                    }, 1000); // Задержка перед скрытием индикатора
                }
            }
            
            this.animationFrame = requestAnimationFrame(detect);
        };
        
        detect();
    }
    
    stop() {
        if (this.animationFrame) {
            cancelAnimationFrame(this.animationFrame);
            this.animationFrame = null;
        }
        
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        
        clearTimeout(this.speakingTimeout);
    }
    
    setThreshold(threshold) {
        this.threshold = threshold;
    }
}
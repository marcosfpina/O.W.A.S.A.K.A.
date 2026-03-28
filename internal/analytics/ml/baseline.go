package ml

import (
	"math"
	"sync"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

// Baseline tracks behavioral patterns and extracts feature vectors for ML.
type Baseline struct {
	mu       sync.RWMutex
	profiles map[string]*hostProfile // keyed by IP
	window   time.Duration           // learning window
	started  time.Time
}

type hostProfile struct {
	eventCounts   map[models.EventType]int
	totalEvents   int
	uniqueDests   map[string]struct{}
	bytesTotal    int64
	firstSeen     time.Time
	lastSeen      time.Time
	hourHistogram [24]int // events per hour of day
}

// NewBaseline creates a behavioral baseline tracker.
func NewBaseline(learningDays int) *Baseline {
	if learningDays == 0 {
		learningDays = 7
	}
	return &Baseline{
		profiles: make(map[string]*hostProfile),
		window:   time.Duration(learningDays) * 24 * time.Hour,
		started:  time.Now(),
	}
}

// Record adds an event to the baseline.
func (b *Baseline) Record(e models.NetworkEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	source := e.Source
	if source == "" {
		return
	}

	p, ok := b.profiles[source]
	if !ok {
		p = &hostProfile{
			eventCounts: make(map[models.EventType]int),
			uniqueDests: make(map[string]struct{}),
			firstSeen:   e.Timestamp,
		}
		b.profiles[source] = p
	}

	p.eventCounts[e.Type]++
	p.totalEvents++
	p.lastSeen = e.Timestamp
	if e.Destination != "" {
		p.uniqueDests[e.Destination] = struct{}{}
	}
	hour := e.Timestamp.Hour()
	p.hourHistogram[hour]++
}

// IsLearning returns true if we're still in the baseline learning phase.
func (b *Baseline) IsLearning() bool {
	return time.Since(b.started) < b.window
}

// FeatureVector extracts a feature vector for a given host's current event.
// Features: [event_rate, dest_diversity, hour_deviation, type_entropy]
func (b *Baseline) FeatureVector(source string, currentHour int) []float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	p, ok := b.profiles[source]
	if !ok {
		return []float64{0, 0, 0, 0}
	}

	// Feature 1: Event rate (events per hour since first seen)
	duration := p.lastSeen.Sub(p.firstSeen).Hours()
	if duration < 1 {
		duration = 1
	}
	eventRate := float64(p.totalEvents) / duration

	// Feature 2: Destination diversity (unique destinations / total events)
	destDiversity := float64(len(p.uniqueDests)) / math.Max(1, float64(p.totalEvents))

	// Feature 3: Hour deviation (how unusual is the current hour for this host)
	hourDev := b.hourDeviation(p, currentHour)

	// Feature 4: Event type entropy
	typeEntropy := b.typeEntropy(p)

	return []float64{eventRate, destDiversity, hourDev, typeEntropy}
}

// TrainingData returns all host feature vectors for forest training.
func (b *Baseline) TrainingData() [][]float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var data [][]float64
	hour := time.Now().Hour()
	for _, p := range b.profiles {
		duration := p.lastSeen.Sub(p.firstSeen).Hours()
		if duration < 1 {
			duration = 1
		}
		eventRate := float64(p.totalEvents) / duration
		destDiv := float64(len(p.uniqueDests)) / math.Max(1, float64(p.totalEvents))
		hourDev := b.hourDeviation(p, hour)
		entropy := b.typeEntropy(p)
		data = append(data, []float64{eventRate, destDiv, hourDev, entropy})
	}
	return data
}

func (b *Baseline) hourDeviation(p *hostProfile, hour int) float64 {
	total := 0
	for _, c := range p.hourHistogram {
		total += c
	}
	if total == 0 {
		return 0
	}
	avg := float64(total) / 24.0
	if avg == 0 {
		return 0
	}
	return math.Abs(float64(p.hourHistogram[hour])-avg) / avg
}

func (b *Baseline) typeEntropy(p *hostProfile) float64 {
	if p.totalEvents == 0 {
		return 0
	}
	var entropy float64
	for _, count := range p.eventCounts {
		prob := float64(count) / float64(p.totalEvents)
		if prob > 0 {
			entropy -= prob * math.Log2(prob)
		}
	}
	return entropy
}
